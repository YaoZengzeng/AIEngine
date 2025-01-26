package router

import (
	aiv1alpha1 "AIEngine/api/v1alpha1"
	"AIEngine/internal/proxy"
	"fmt"
	"strings"
)

type ModelRouter interface {
	Route(model string, message string) (string, error)

	UpdateRoute(models []string, rules []*aiv1alpha1.Rule) error
	DeleteRoute(models []string) error
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
		return "", fmt.Errorf("not found route rules for model %s", model)
	}

	rule, err := m.selectRule(model, rules)
	if err != nil {
		return "", fmt.Errorf("failed to select route rule: %v", err)
	}

	dst, err := m.selectDestination(rule.Route)
	if err != nil {
		return "", fmt.Errorf("failed to select destination: %v", err)
	}

	s := strings.Split(dst.Destination.Model, "/")
	if len(s) >= 3 {
		return "", fmt.Errorf("the format of backend model should be \"model\" or \"provider/model\"")
	}

	var backendProvider, backendModel string
	if len(s) == 1 {
		backendModel = s[0]
	} else {
		backendProvider = s[0]
		backendModel = s[1]
	}

	return m.proxy.Proxy(dst.Destination.Host, backendProvider, backendModel, message)
}

func (m *modelRouterImpl) selectRule(model string, rules []*aiv1alpha1.Rule) (*aiv1alpha1.Rule, error) {
	// For POC, directly use the first rule of the first route.
	if len(rules) == 0 || len(rules[0].Route) == 0 {
		return nil, fmt.Errorf("empty rules or route")
	}

	return rules[0], nil
}

func (m *modelRouterImpl) selectDestination(routes []*aiv1alpha1.RouteDestination) (*aiv1alpha1.RouteDestination, error) {
	return routes[0], nil
}

func (m *modelRouterImpl) UpdateRoute(models []string, rules []*aiv1alpha1.Rule) error {
	fmt.Printf("UpdateRoute models: %v\n", models)

	for _, model := range models {
		m.routes[model] = rules
	}

	return nil
}

func (m *modelRouterImpl) DeleteRoute(models []string) error {
	fmt.Printf("DeleteRoute models: %v\n", models)

	for _, model := range models {
		// TODO: consider a mode configured in multiple VirtualModel?
		delete(m.routes, model)
	}

	return nil
}
