package router

import (
	"fmt"
	"math/rand"
	"time"

	aiv1alpha1 "AIEngine/api/v1alpha1"
	"AIEngine/internal/proxy"
)

type ModelRouter interface {
	Route(model string) (*aiv1alpha1.TargetModel, error)

	UpdateRoute(model string, rules []*aiv1alpha1.Rule) error
	DeleteRoute(model string) error
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

func (m *modelRouterImpl) Route(model string) (*aiv1alpha1.TargetModel, error) {
	rules, ok := m.routes[model]
	if !ok {
		return nil, fmt.Errorf("not found route rules for model %s", model)
	}

	rule, err := m.selectRule(model, rules)
	if err != nil {
		return nil, fmt.Errorf("failed to select route rule: %v", err)
	}

	dst, err := m.selectDestination(rule.TargetModels)
	if err != nil {
		return nil, fmt.Errorf("failed to select destination: %v", err)
	}

	return dst, nil
}

func (m *modelRouterImpl) selectRule(model string, rules []*aiv1alpha1.Rule) (*aiv1alpha1.Rule, error) {
	// For POC, directly use the first rule.
	if len(rules) == 0 || len(rules[0].TargetModels) == 0 {
		return nil, fmt.Errorf("empty rules or target models")
	}

	return rules[0], nil
}

func (m *modelRouterImpl) selectDestination(targets []*aiv1alpha1.TargetModel) (*aiv1alpha1.TargetModel, error) {
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

func (m *modelRouterImpl) UpdateRoute(model string, rules []*aiv1alpha1.Rule) error {
	fmt.Printf("UpdateRoute model: %v\n", model)

	m.routes[model] = rules

	return nil
}

func (m *modelRouterImpl) DeleteRoute(model string) error {
	fmt.Printf("DeleteRoute model: %v\n", model)

	delete(m.routes, model)

	return nil
}
