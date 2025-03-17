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
	"fmt"
	"sync"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	aiv1alpha1 "AIEngine/api/v1alpha1"
)

// ModelServerReconciler reconciles a ModelServer object
type ModelServerReconciler struct {
	client.Client
	Scheme *runtime.Scheme

	PodMapping sync.Map
}

func (r *ModelServerReconciler) UpdatePodMapping(key types.NamespacedName, pods []corev1.Pod) {
	fmt.Printf("ModelServer is %v, pods: \n", key)
	for _, pod := range pods {
		fmt.Printf("%v\n", pod.Name)
	}

	r.PodMapping.Store(key, pods)
}

func (r *ModelServerReconciler) GetPodsFromModel(key types.NamespacedName) []corev1.Pod {
	if val, ok := r.PodMapping.Load(key); ok {
		return val.([]corev1.Pod)
	}

	return nil
}

// +kubebuilder:rbac:groups=ai.kmesh.net,resources=ModelServers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=ai.kmesh.net,resources=ModelServers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=ai.kmesh.net,resources=ModelServers/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ModelServer object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.1/pkg/reconcile
func (r *ModelServerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	ModelServer := &aiv1alpha1.ModelServer{}
	if err := r.Get(ctx, req.NamespacedName, ModelServer); err != nil {
		r.PodMapping.Delete(req.NamespacedName)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{MatchLabels: ModelServer.Spec.WorkloadSelector.MatchLabels})
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("invalid selector: %v", err)
	}

	var pods corev1.PodList
	if err := r.List(ctx, &pods, client.InNamespace(req.Namespace), client.MatchingLabelsSelector{Selector: selector}); err != nil {
		return ctrl.Result{}, err
	}

	r.UpdatePodMapping(req.NamespacedName, pods.Items)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ModelServerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&aiv1alpha1.ModelServer{}).
		Watches(&corev1.Pod{}, handler.EnqueueRequestsFromMapFunc(handler.MapFunc(r.podEventHandler)), builder.WithPredicates(predicate.LabelChangedPredicate{})).
		Named("ModelServer").
		Complete(r)
}

func (r *ModelServerReconciler) podEventHandler(ctx context.Context, obj client.Object) []reconcile.Request {
	pod := obj.(*corev1.Pod)
	ModelServers := &aiv1alpha1.ModelServerList{}
	if err := r.List(ctx, ModelServers, client.InNamespace(pod.Namespace)); err != nil {
		return nil
	}

	var requests []reconcile.Request
	for _, tm := range ModelServers.Items {
		selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{MatchLabels: tm.Spec.WorkloadSelector.MatchLabels})
		if err != nil || !selector.Matches(labels.Set(pod.Labels)) {
			continue
		}
		requests = append(requests, reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      tm.Name,
				Namespace: tm.Namespace,
			},
		})
	}

	return requests
}
