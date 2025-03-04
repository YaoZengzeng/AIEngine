/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	envoy_api_v3_core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	envoy_service_proc_v3 "github.com/envoyproxy/go-control-plane/envoy/service/ext_proc/v3"
	v32 "github.com/envoyproxy/go-control-plane/envoy/type/v3"
	"github.com/tmc/langchaingo/llms"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	aiv1alpha1 "AIEngine/api/v1alpha1"
	"AIEngine/internal/limiter"
	"AIEngine/internal/router"
)

// VirtualModelReconciler reconciles a VirtualModel object
type VirtualModelReconciler struct {
	client.Client
	Scheme *runtime.Scheme

	ResourceToModels map[string][]string

	ModelRouter router.ModelRouter

	RateLimiter limiter.RateLimiter
}

// +kubebuilder:rbac:groups=ai.kmesh.net,resources=virtualmodels,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=ai.kmesh.net,resources=virtualmodels/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=ai.kmesh.net,resources=virtualmodels/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the VirtualModel object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.1/pkg/reconcile
func (r *VirtualModelReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	var vm aiv1alpha1.VirtualModel
	if err := r.Get(ctx, req.NamespacedName, &vm); err != nil {
		if apierrors.IsNotFound(err) {
			models := r.GetFromResourceMap(req.NamespacedName.String())
			if err := r.ModelRouter.DeleteRoute(models); err != nil {
				fmt.Printf("failed to delete route: %v", err)
				return ctrl.Result{}, nil
			}
		}

		fmt.Printf("unable to fetch VirtualModel: %v", err)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	log.V(1).Info("Get VirtualModel", "models", vm.Spec.Models)

	r.SetForResourceMap(req.NamespacedName.String(), vm.Spec.Models)

	if err := r.ModelRouter.UpdateRoute(vm.Spec.Models, vm.Spec.Rules); err != nil {
		log.Error(err, "failed to update ModelRouter")
		return ctrl.Result{}, err
	}

	if vm.Spec.RateLimit != nil {
		for _, model := range vm.Spec.Models {
			if err := r.RateLimiter.UpdateConfig(&limiter.RateLimitConfig{Model: model, TokensPerUnit: vm.Spec.RateLimit.TokensPerUnit, Unit: string(vm.Spec.RateLimit.Unit)}); err != nil {
				log.Error(err, "failed to update config of RateLimiter")
				return ctrl.Result{}, err
			}
		}
	}

	return ctrl.Result{}, nil
}

func (r *VirtualModelReconciler) GetFromResourceMap(namespacedName string) []string {
	return r.ResourceToModels[namespacedName]
}

func (r *VirtualModelReconciler) SetForResourceMap(namespacedName string, models []string) {
	r.ResourceToModels[namespacedName] = models
}

func (r *VirtualModelReconciler) Process(srv envoy_service_proc_v3.ExternalProcessor_ProcessServer) error {
	fmt.Printf("--- Calling VirtualModelReconciler Process\n")

	host := ""
	ctx := srv.Context()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		req, err := srv.Recv()
		if err == io.EOF {
			fmt.Printf("--- Recv() get EOF, Exit VirtualModelReconciler Process")
			return nil
		}
		if err != nil {
			return status.Errorf(codes.Unknown, "cannot receive stream request: %v", err)
		}

		resp := &envoy_service_proc_v3.ProcessingResponse{}
		switch v := req.Request.(type) {
		case *envoy_service_proc_v3.ProcessingRequest_RequestHeaders:
			fmt.Printf("Handle Request Headers\n")

			rhq := &envoy_service_proc_v3.HeadersResponse{
				Response: &envoy_service_proc_v3.CommonResponse{
					HeaderMutation: &envoy_service_proc_v3.HeaderMutation{
						SetHeaders: []*envoy_api_v3_core.HeaderValueOption{
							{
								Header: &envoy_api_v3_core.HeaderValue{
									Key:      "x-request-ext-processed",
									RawValue: []byte("true"),
								},
							},
						},
					},
				},
			}

			resp = &envoy_service_proc_v3.ProcessingResponse{
				Response: &envoy_service_proc_v3.ProcessingResponse_RequestHeaders{
					RequestHeaders: rhq,
				},
			}
		case *envoy_service_proc_v3.ProcessingRequest_RequestBody:
			fmt.Printf("Handle Request Body\n")

			rbq := &envoy_service_proc_v3.BodyResponse{
				Response: &envoy_service_proc_v3.CommonResponse{},
			}

			resp = &envoy_service_proc_v3.ProcessingResponse{
				Response: &envoy_service_proc_v3.ProcessingResponse_RequestBody{
					RequestBody: rbq,
				},
			}

			if v.RequestBody != nil {
				body := v.RequestBody.GetBody()
				fmt.Printf("Request BODY: %s\nEnd of Stream is %v\n", string(body), v.RequestBody.EndOfStream)

				var req Request

				err := json.Unmarshal(body, &req)
				if err != nil {
					fmt.Printf("Failed to unmarshal req")
					break
				}

				tokenCount := llms.CountTokens("", req.Prompt)
				fmt.Printf("Token Count is %d, DoLimit model: %s\n", tokenCount, req.Model)
				if allowed := r.RateLimiter.DoLimit(host, req.Model, tokenCount); !allowed {
					fmt.Printf("Trigger Rate Limit\n")
					resp = &envoy_service_proc_v3.ProcessingResponse{
						Response: &envoy_service_proc_v3.ProcessingResponse_ImmediateResponse{
							ImmediateResponse: &envoy_service_proc_v3.ImmediateResponse{
								Status: &v32.HttpStatus{Code: v32.StatusCode_TooManyRequests},
							},
						},
					}

					if err := srv.Send(resp); err != nil {
						fmt.Printf("send error %v", err)
					}

					continue
				}

				fmt.Printf("VirtualModel host: %s, model: %s\n", host, req.Model)

				if _, err := r.ModelRouter.Route(req.Model, req.Prompt); err != nil {
					resp = &envoy_service_proc_v3.ProcessingResponse{
						Response: &envoy_service_proc_v3.ProcessingResponse_ImmediateResponse{
							ImmediateResponse: &envoy_service_proc_v3.ImmediateResponse{
								Status: &v32.HttpStatus{Code: v32.StatusCode_BadRequest},
							},
						},
					}

					if err := srv.Send(resp); err != nil {
						fmt.Printf("send error %v", err)
					}

					continue
				}

				rhq := &envoy_service_proc_v3.BodyResponse{
					Response: &envoy_service_proc_v3.CommonResponse{
						HeaderMutation: &envoy_service_proc_v3.HeaderMutation{
							SetHeaders: []*envoy_api_v3_core.HeaderValueOption{
								{
									Header: &envoy_api_v3_core.HeaderValue{
										Key:      "kmesh-selected-ip",
										RawValue: []byte("10.244.0.8"),
									},
								},
							},
						},
					},
				}

				// Return Immediate response anyway
				resp = &envoy_service_proc_v3.ProcessingResponse{
					Response: &envoy_service_proc_v3.ProcessingResponse_ResponseBody{
						ResponseBody: rhq,
					},
				}
			}
		case *envoy_service_proc_v3.ProcessingRequest_RequestTrailers:
			fmt.Printf("Handle Request Trailers\n")
		case *envoy_service_proc_v3.ProcessingRequest_ResponseHeaders:
			fmt.Printf("Handle Response Headers\n")

			if v.ResponseHeaders != nil {
				hdrs := v.ResponseHeaders.Headers.GetHeaders()
				for _, hdr := range hdrs {
					fmt.Printf("RESPONSE HEADER: %s:%s\n", hdr.Key, hdr.Value)
				}
			}

			rhq := &envoy_service_proc_v3.HeadersResponse{
				Response: &envoy_service_proc_v3.CommonResponse{
					HeaderMutation: &envoy_service_proc_v3.HeaderMutation{
						SetHeaders: []*envoy_api_v3_core.HeaderValueOption{
							{
								Header: &envoy_api_v3_core.HeaderValue{
									Key:      "x-response-ext-processed",
									RawValue: []byte("true"),
								},
							},
						},
					},
				},
			}
			resp = &envoy_service_proc_v3.ProcessingResponse{
				Response: &envoy_service_proc_v3.ProcessingResponse_ResponseHeaders{
					ResponseHeaders: rhq,
				},
			}
		case *envoy_service_proc_v3.ProcessingRequest_ResponseBody:
			fmt.Printf("Handle Response Body\n")

			if v.ResponseBody != nil {
				body := v.ResponseBody.GetBody()
				fmt.Printf("Response BODY: %s\nEnd of Stream is %v\n", string(body), v.ResponseBody.EndOfStream)
			}

			if v.ResponseBody.EndOfStream {
				return nil
			}

			rbq := &envoy_service_proc_v3.BodyResponse{
				Response: &envoy_service_proc_v3.CommonResponse{},
			}

			resp = &envoy_service_proc_v3.ProcessingResponse{
				Response: &envoy_service_proc_v3.ProcessingResponse_ResponseBody{
					ResponseBody: rbq,
				},
			}
		case *envoy_service_proc_v3.ProcessingRequest_ResponseTrailers:
			fmt.Printf("Handle Response Trailers\n")
		default:
			fmt.Printf("Unknown Request type %v\n", v)
		}

		if err := srv.Send(resp); err != nil {
			fmt.Printf("send error %v", err)
		}
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *VirtualModelReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&aiv1alpha1.VirtualModel{}).
		Named("virtualmodel").
		Complete(r)
}
