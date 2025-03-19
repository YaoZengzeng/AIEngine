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
	// `model` in the LLM request, it could be a base model name, lora adapter name or even
	// a virtual model name. This field is used to match scenarios other than model adapter name and
	// this field could be empty, but it and  `ModelAdapters` can't both be empty.
	ModelName string `json:"modelName"`

	// `model` in the LLM request could be lora adapter name,
	// here is a list of Lora Adapter Names to match.
	LoraAdapters []string `json:"loraAdapters,omitemtpy"`

	// An ordered list of route rules for LLM traffic. The first rule
	// matching an incoming request will be used.
	Rules []*Rule `json:"rules"`

	// Rate limit for `ModelName`
	RateLimit *RateLimit `json:"rateLimit,omitempty"`

	// Auth *Auth `json:"auth,omitempty"`
}

type Rule struct {
	Name string `json:"name,omitempty"`
	// If no `modelMatch` is specified, it will be default route.
	ModelMatche  *ModelMatch    `json:"modelMatch,omitempty"`
	TargetModels []*TargetModel `json:"targetModels"`
}

type ModelMatch struct {
	// Header to match: prefix, exact, regex
	Headers map[string]*StringMatch `json:"headers,omitempty"`

	// URI to match: prefix, exact, regex
	Uri *StringMatch `json:"uri,omitempty"`

	// More if necessary.
}

type TargetModel struct {
	ModelServerName string  `json:"modelServerName"`
	Weight          *uint32 `json:"weight,omitempty"`
}

type RateLimit struct {
	TokensPerUnit uint32        `json:"tokensPerUnit"`
	Unit          RateLimitUnit `json:"unit"`
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

type StringMatch struct {
	Exact  *string `json:"exact,omitempty"`
	Prefix *string `json:"prefix,omitempty"`
	Regex  *string `json:"regex,omitempty"`
}

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
