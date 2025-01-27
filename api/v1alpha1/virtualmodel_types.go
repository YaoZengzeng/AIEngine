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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// VirtualModelSpec defines the desired state of VirtualModel.
type VirtualModelSpec struct {
	Models []string `json:"models,omitempty"`

	Rules []*Rule `json:"rules"`
}

// VirtualModelStatus defines the observed state of VirtualModel.
type VirtualModelStatus struct {
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// VirtualModel is the Schema for the virtualmodels API.
type VirtualModel struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VirtualModelSpec   `json:"spec,omitempty"`
	Status VirtualModelStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// VirtualModelList contains a list of VirtualModel.
type VirtualModelList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VirtualModel `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VirtualModel{}, &VirtualModelList{})
}

type Rule struct {
	Name  string              `json:"name,omitempty"`
	Match []*MatchCondition   `json:"match,omitempty"`
	Route []*RouteDestination `json:"route"`

	Timeout *metav1.Duration `json:"timeout,omitempty"`
	Retries *Retry           `json:"retries,omitempty"`
}

type MatchCondition struct {
	// Implement if necessary.
}

type RouteDestination struct {
	Destination *Destination `json:"destination"`
	Weight      *uint32      `json:"weight,omitempty"`
}

type Destination struct {
	Host  string `json:"host"`
	Model string `json:"model"`
}

type Retry struct {
	Attempts int32 `json:"attempts"`
}
