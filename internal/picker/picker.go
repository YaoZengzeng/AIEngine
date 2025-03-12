package picker

import (
	aiv1alpha1 "AIEngine/api/v1alpha1"
)

type EndpointPicker interface {
	PickEndpoint(*aiv1alpha1.Destination) (*Endpoint, error)
}

type Endpoint struct {
	Address string
}

type endpointPickerImpl struct {
}

func NewEndpointPicker() (EndpointPicker, error) {
	return &endpointPickerImpl{}, nil
}

func (e *endpointPickerImpl) PickEndpoint(*aiv1alpha1.Destination) (*Endpoint, error) {
	return &Endpoint{Address: "10.244.0.8"}, nil
}
