package router

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	aiv1alpha1 "AIEngine/api/v1alpha1"
	"AIEngine/internal/proxy"
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
	// fmt.Printf("Route model %s, message %s\n", model, message)

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

	fmt.Printf("-- Route to backend host %s, backend provider %s, backend model %s\n", dst.Destination.Host, backendProvider, backendModel)

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
	weightedSlice, err := toWeightedSlice(routes)
	if err != nil {
		return nil, err
	}

	index := selectFromWeightedSlice(weightedSlice)

	return routes[index], nil
}

func toWeightedSlice(routes []*aiv1alpha1.RouteDestination) ([]uint32, error) {
	var isWeighted bool
	if routes[0].Weight != nil {
		isWeighted = true
	}

	res := make([]uint32, len(routes))

	for i, route := range routes {
		if (isWeighted && route.Weight == nil) || (!isWeighted && route.Weight != nil) {
			return nil, fmt.Errorf("the weight field in routes must be either fully specified or not specified")
		}

		if isWeighted {
			res[i] = *route.Weight
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
