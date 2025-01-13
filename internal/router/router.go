package router

import (
	aiv1alpha1 "AIEngine/api/v1alpha1"
	"AIEngine/internal/proxy"
	"fmt"
	"strings"
)

type ModelRouter interface {
	Route(model string, message string) (string, error)

	UpdateRoute(vm *aiv1alpha1.VirtualModel) error
	DeleteRoute(vm *aiv1alpha1.VirtualModel) error
}

type modelRouterImpl struct {
	routes map[string][]*aiv1alpha1.Rule

	proxy proxy.Proxy
}

func NewModelRouter() (ModelRouter, error) {
	proxy, err := proxy.NewProxy()
	if err != nil {
		return nil, fmt.Errorf("failed to construct proxy")
	}
	return &modelRouterImpl{routes: make(map[string][]*aiv1alpha1.Rule), proxy: proxy}, nil
}

func (m *modelRouterImpl) Route(model string, message string) (string, error) {
	fmt.Printf("Route model %s, message %s\n", model, message)

	rules, ok := m.routes[model]
	if !ok {
		return "", fmt.Errorf("Not found route rules for model %s", model)
	}

	// For POC, directly use the first rule of the first route.
	if len(rules) == 0 || len(rules[0].Route) == 0 {
		return "", fmt.Errorf("Empty rules or route")
	}

	host := rules[0].Route[0].Destination.Host

	s := strings.Split(rules[0].Route[0].Destination.Model, "/")
	if len(s) >= 3 {
		return "", fmt.Errorf("The format of backend model should be \"model\" or \"provider/model\"")
	}

	var backendProvider, backendModel string
	if len(s) == 1 {
		backendModel = s[0]
	} else {
		backendProvider = s[0]
		backendModel = s[1]
	}

	return m.proxy.Proxy(host, backendProvider, backendModel, message)
}

func (m *modelRouterImpl) UpdateRoute(vm *aiv1alpha1.VirtualModel) error {
	fmt.Printf("UpdateRoute models: %v\n", vm.Spec.Models)

	for _, model := range vm.Spec.Models {
		m.routes[model] = vm.Spec.Rules
	}

	return nil
}

func (m *modelRouterImpl) DeleteRoute(vm *aiv1alpha1.VirtualModel) error {
	fmt.Printf("DeleteRoute models: %v\n", vm.Spec.Models)
	return nil
}
