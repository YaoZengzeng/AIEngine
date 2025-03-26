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
	"math/rand"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	aiv1alpha1 "AIEngine/api/v1alpha1"
)

// ModelRouteReconciler reconciles a ModelRoute object
type ModelRouteReconciler struct {
	client.Client
	Scheme *runtime.Scheme

	ResourceToModels map[string]string
	Routes           map[string][]*aiv1alpha1.Rule
}

type Request struct {
	Model string `json:"model"`

	Prompt string `json:"prompt"`
}

// +kubebuilder:rbac:groups=ai.kmesh.net,resources=ModelRoutes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=ai.kmesh.net,resources=ModelRoutes/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=ai.kmesh.net,resources=ModelRoutes/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ModelRoute object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.1/pkg/reconcile
func (m *ModelRouteReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	var mr aiv1alpha1.ModelRoute
	if err := m.Get(ctx, req.NamespacedName, &mr); err != nil {
		if apierrors.IsNotFound(err) {
			model := m.GetFromResourceMap(req.NamespacedName.String())
			if err := m.DeleteRoute(model); err != nil {
				fmt.Printf("failed to delete route: %v", err)
				return ctrl.Result{}, nil
			}
		}

		fmt.Printf("unable to fetch ModelRoute: %v", err)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	log.V(1).Info("Get ModelRoute", "model", mr.Spec.ModelName)

	m.SetForResourceMap(req.NamespacedName.String(), mr.Spec.ModelName)

	if err := m.UpdateRoute(mr.Spec.ModelName, mr.Spec.Rules); err != nil {
		log.Error(err, "failed to update ModelRouter")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (m *ModelRouteReconciler) GetFromResourceMap(namespacedName string) string {
	return m.ResourceToModels[namespacedName]
}

func (m *ModelRouteReconciler) SetForResourceMap(namespacedName string, model string) {
	m.ResourceToModels[namespacedName] = model
}

func (m *ModelRouteReconciler) Process(model string) (string, error) {
	rules, ok := m.Routes[model]
	if !ok {
		return "", fmt.Errorf("not found route rules for model %s", model)
	}

	rule, err := m.selectRule(model, rules)
	if err != nil {
		return "", fmt.Errorf("failed to select route rule: %v", err)
	}

	dst, err := m.selectDestination(rule.TargetModels)
	if err != nil {
		return "", fmt.Errorf("failed to select destination: %v", err)
	}

	// TODO: fix namespace.
	return types.NamespacedName{Namespace: "default", Name: dst.ModelServerName}.String(), nil
}

func (m *ModelRouteReconciler) selectRule(model string, rules []*aiv1alpha1.Rule) (*aiv1alpha1.Rule, error) {
	// For POC, directly use the first rule.
	if len(rules) == 0 || len(rules[0].TargetModels) == 0 {
		return nil, fmt.Errorf("empty rules or target models")
	}

	return rules[0], nil
}

func (m *ModelRouteReconciler) selectDestination(targets []*aiv1alpha1.TargetModel) (*aiv1alpha1.TargetModel, error) {
	weightedSlice, err := toWeightedSlice(targets)
	if err != nil {
		return nil, err
	}

	index := selectFromWeightedSlice(weightedSlice)

	return targets[index], nil
}

func toWeightedSlice(targets []*aiv1alpha1.TargetModel) ([]uint32, error) {
	var isWeighted bool
	if targets[0].Weight != nil {
		isWeighted = true
	}

	res := make([]uint32, len(targets))

	for i, target := range targets {
		if (isWeighted && target.Weight == nil) || (!isWeighted && target.Weight != nil) {
			return nil, fmt.Errorf("the weight field in targetModel must be either fully specified or not specified")
		}

		if isWeighted {
			res[i] = *target.Weight
		} else {
			// If weight is not specified, set to 1.
			res[i] = 1
		}
	}

	return res, nil
}

func selectFromWeightedSlice(weights []uint32) int {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	totalWeight := 0
	for _, weight := range weights {
		totalWeight += int(weight)
	}

	randomNum := rng.Intn(totalWeight)

	for i, weight := range weights {
		randomNum -= int(weight)
		if randomNum < 0 {
			return i
		}
	}

	return 0
}

func (m *ModelRouteReconciler) UpdateRoute(model string, rules []*aiv1alpha1.Rule) error {
	fmt.Printf("UpdateRoute model: %v\n", model)

	m.Routes[model] = rules

	return nil
}

func (m *ModelRouteReconciler) DeleteRoute(model string) error {
	fmt.Printf("DeleteRoute model: %v\n", model)

	delete(m.Routes, model)

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (m *ModelRouteReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&aiv1alpha1.ModelRoute{}).
		Named("ModelRoute").
		Complete(m)
}
