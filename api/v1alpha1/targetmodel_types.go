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

// TargetModelSpec defines the desired state of TargetModel.
type TargetModelSpec struct {
	Name             string            `json:"name"`
	Hostname         string            `json:"hostname,omitempty"`
	WorkloadSelector *WorkloadSelector `json:"workloadSelector,omitempty"`
	TrafficPolicy    *TrafficPolicy    `json:"trafficPolicy,omitempty"`
	Subsets          []*Subset         `json:"subsets,omitempty"`
	Loras            []*Lora           `json:"loras,omitempty"`
}

type WorkloadSelector struct {
	MatchLabels map[string]string `json:"matchLabels,omitempty"`
}

type TrafficPolicy struct {
	LoadBalancer *LoadBalancerSettings `json:"loadBalancer,omitempty"`
}

type LoadBalancerSettings struct {
	Simple SimpleLBPolicy `json:"simple,omitempty"`
}

type SimpleLBPolicy int32

const (
	RANDOM      SimpleLBPolicy = 0
	ROUND_ROBIN SimpleLBPolicy = 1
)

type Subset struct {
	Name          string            `json:"name"`
	Labels        map[string]string `json:"labels"`
	TrafficPolicy *TrafficPolicy    `json:"trafficPolicy,omitempty"`
}

type Lora struct {
	Name          string         `json:"name"`
	TrafficPolicy *TrafficPolicy `json:"trafficPolicy,omitempty"`
}

// TargetModelStatus defines the observed state of TargetModel.
type TargetModelStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// TargetModel is the Schema for the targetmodels API.
type TargetModel struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TargetModelSpec   `json:"spec,omitempty"`
	Status TargetModelStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// TargetModelList contains a list of TargetModel.
type TargetModelList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TargetModel `json:"items"`
}

func init() {
	SchemeBuilder.Register(&TargetModel{}, &TargetModelList{})
}
