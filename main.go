package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"time"

	"google.golang.org/grpc/credentials/insecure"

	"github.com/tmc/langchaingo/llms"
	"golang.org/x/time/rate"

	envoy_api_v3_core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	envoy_service_proc_v3 "github.com/envoyproxy/go-control-plane/envoy/service/ext_proc/v3"
	v32 "github.com/envoyproxy/go-control-plane/envoy/type/v3"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type AIEngine struct {
	rateLimiter map[string]*rate.Limiter
}

type Request struct {
	Model string `json:"model"`

	Prompt string `json:"prompt"`
}

var (
	port int
)

func main() {
	flag.IntVar(&port, "port", 9002, "gRPC port")
	flag.Parse()

	log.Printf("Start AI Engine\n")

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	gs := grpc.NewServer()
	envoy_service_proc_v3.RegisterExternalProcessorServer(gs, &AIEngine{rateLimiter: make(map[string]*rate.Limiter)})

	go func() {
		err = gs.Serve(lis)
		if err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	http.HandleFunc("/healthz", healthCheckHandler)
	err = http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

// used by k8s readiness probes
// makes a processing request to check if the processor service is healthy
func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}

	conn, err := grpc.Dial("localhost:9002", opts...)
	if err != nil {
		log.Fatalf("Could not connect: %v", err)
	}
	client := envoy_service_proc_v3.NewExternalProcessorClient(conn)

	processor, err := client.Process(context.Background())
	if err != nil {
		log.Fatalf("Could not check: %v", err)
	}

	err = processor.Send(&envoy_service_proc_v3.ProcessingRequest{
		Request: &envoy_service_proc_v3.ProcessingRequest_RequestHeaders{
			RequestHeaders: &envoy_service_proc_v3.HttpHeaders{},
		},
	})
	if err != nil {
		log.Fatalf("Could not check: %v", err)
	}

	response, err := processor.Recv()
	if err != nil {
		log.Fatalf("Could not check: %v", err)
	}

	if response != nil && response.GetRequestHeaders().Response.Status == envoy_service_proc_v3.CommonResponse_CONTINUE {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
	}
}

func (s *AIEngine) Process(srv envoy_service_proc_v3.ExternalProcessor_ProcessServer) error {
	ctx := srv.Context()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		req, err := srv.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return status.Errorf(codes.Unknown, "cannot receive stream request: %v", err)
		}

		resp := &envoy_service_proc_v3.ProcessingResponse{}
		switch v := req.Request.(type) {
		case *envoy_service_proc_v3.ProcessingRequest_RequestHeaders:
			log.Printf("Handle Request Headers\n")

			xrch := ""
			if v.RequestHeaders != nil {
				hdrs := v.RequestHeaders.Headers.GetHeaders()
				for _, hdr := range hdrs {
					log.Printf("REQUEST HEADER: %s:%s\n", hdr.Key, hdr.Value)
					if hdr.Key == "x-request-client-header" {
						xrch = string(hdr.RawValue)
					}
				}
			}

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

			if xrch != "" {
				rhq.Response.HeaderMutation.SetHeaders = append(rhq.Response.HeaderMutation.SetHeaders,
					&envoy_api_v3_core.HeaderValueOption{
						Header: &envoy_api_v3_core.HeaderValue{
							Key:      "x-request-client-header",
							RawValue: []byte("mutated"),
						},
					})
				rhq.Response.HeaderMutation.SetHeaders = append(rhq.Response.HeaderMutation.SetHeaders,
					&envoy_api_v3_core.HeaderValueOption{
						Header: &envoy_api_v3_core.HeaderValue{
							Key:      "x-request-client-header-received",
							RawValue: []byte(xrch),
						},
					})
			}

			resp = &envoy_service_proc_v3.ProcessingResponse{
				Response: &envoy_service_proc_v3.ProcessingResponse_RequestHeaders{
					RequestHeaders: rhq,
				},
			}
			break
		case *envoy_service_proc_v3.ProcessingRequest_RequestBody:
			log.Printf("Handle Request Body")

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
				log.Printf("Request BODY: %s", string(body))

				var req Request

				err := json.Unmarshal(body, &req)
				if err != nil {
					log.Printf("Failed to unmarshal req")
					break
				}

				if _, ok := s.rateLimiter[req.Model]; !ok {
					s.rateLimiter[req.Model] = rate.NewLimiter(rate.Every(time.Minute), 10)
				}

				tokenCount := llms.CountTokens("", req.Prompt)
				log.Printf("Token Count is %d", tokenCount)

				if allowed := s.rateLimiter[req.Model].AllowN(time.Now(), tokenCount); !allowed {
					log.Printf("Trigger Rate Limit")
					resp = &envoy_service_proc_v3.ProcessingResponse{
						Response: &envoy_service_proc_v3.ProcessingResponse_ImmediateResponse{
							ImmediateResponse: &envoy_service_proc_v3.ImmediateResponse{
								Status: &v32.HttpStatus{Code: v32.StatusCode_TooManyRequests},
							},
						},
					}
				}
			}
			break
		case *envoy_service_proc_v3.ProcessingRequest_RequestTrailers:
			log.Printf("Handle Request Trailers")
			break
		case *envoy_service_proc_v3.ProcessingRequest_ResponseHeaders:
			log.Printf("Handle Response Headers")

			if v.ResponseHeaders != nil {
				hdrs := v.ResponseHeaders.Headers.GetHeaders()
				for _, hdr := range hdrs {
					log.Printf("RESPONSE HEADER: %s:%s\n", hdr.Key, hdr.Value)
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
			break
		case *envoy_service_proc_v3.ProcessingRequest_ResponseBody:
			log.Printf("Handle Response Body")

			if v.ResponseBody != nil {
				body := v.ResponseBody.GetBody()
				log.Printf("Response BODY: %s", string(body))
			}

			rbq := &envoy_service_proc_v3.BodyResponse{
				Response: &envoy_service_proc_v3.CommonResponse{},
			}

			resp = &envoy_service_proc_v3.ProcessingResponse{
				Response: &envoy_service_proc_v3.ProcessingResponse_ResponseBody{
					ResponseBody: rbq,
				},
			}
			break
		case *envoy_service_proc_v3.ProcessingRequest_ResponseTrailers:
			log.Printf("Handle Response Trailers")
			break
		default:
			log.Printf("Unknown Request type %v\n", v)
		}
		if err := srv.Send(resp); err != nil {
			log.Printf("send error %v", err)
		}
	}
}
