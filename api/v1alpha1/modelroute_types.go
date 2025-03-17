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

// ModelRouteSpec defines the desired state of ModelRoute.
type ModelRouteSpec struct {
	ModelName string `json:"modelName"`

	Rules []*Rule `json:"rules"`

	RateLimit *RateLimit `json:"rateLimit,omitempty"`

	// Auth *Auth `json:"auth,omitempty"`
}

type Rule struct {
	Name         string         `json:"name,omitempty"`
	ModelMatches []*ModelMatch  `json:"modelMatches,omitempty"`
	TargetModels []*TargetModel `json:"targetModels"`
}

type ModelMatch struct {
	// Implement the following, if necessary:
	// Header: prefix, exact, regex
	// Url
	// Path
}

type TargetModel struct {
	Name   string  `json:"name"`
	Subset *string `json:"subset,omitempty"`
	Lora   *string `json:"lora,omitempty"`
	Weight *uint32 `json:"weight,omitempty"`
}

type RateLimit struct {
	TokensPerUnit uint32        `json:"tokensPerUnit"`
	Unit          RateLimitUnit `json:"unit"`
	Model         string        `json:"model,omitempty"`
}

// +kubebuilder:validation:Enum=second;minute;hour;day;month
type RateLimitUnit string

const (
	Second RateLimitUnit = "second"
	Minute RateLimitUnit = "minute"
	Hour   RateLimitUnit = "hour"
	Day    RateLimitUnit = "day"
	Month  RateLimitUnit = "month"
)

// ModelRouteStatus defines the observed state of ModelRoute.
type ModelRouteStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// ModelRoute is the Schema for the modelroutes API.
type ModelRoute struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ModelRouteSpec   `json:"spec,omitempty"`
	Status ModelRouteStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ModelRouteList contains a list of ModelRoute.
type ModelRouteList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ModelRoute `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ModelRoute{}, &ModelRouteList{})
}
