package router

import (
	aiv1alpha1 "AIEngine/api/v1alpha1"
	"fmt"
)

type ModelRouter interface {
	Route(model string, message string) (string, error)

	UpdateRoute(vm *aiv1alpha1.VirtualModel) error
	DeleteRoute(vm *aiv1alpha1.VirtualModel) error
}

type modelRouterImpl struct {
}

func NewModelRouter() (ModelRouter, error) {
	return &modelRouterImpl{}, nil
}

func (m *modelRouterImpl) Route(model string, message string) (string, error) {
	fmt.Printf("Route model %s, message %s\n", model, message)
	return "", nil
}

func (m *modelRouterImpl) UpdateRoute(vm *aiv1alpha1.VirtualModel) error {
	fmt.Printf("UpdateRoute models: %v\n", vm.Spec.Models)
	return nil
}

func (m *modelRouterImpl) DeleteRoute(vm *aiv1alpha1.VirtualModel) error {
	fmt.Printf("DeleteRoute models: %v\n", vm.Spec.Models)
	return nil
}
