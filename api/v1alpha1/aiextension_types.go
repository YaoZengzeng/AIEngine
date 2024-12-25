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

// AIExtensionSpec defines the desired state of AIExtension.
type AIExtensionSpec struct {
	Hostname string  `json:"hostname"`
	Options  Options `json:"options"`
}

type Options struct {
	RateLimits []RateLimit `json:"rateLimits,omitempty"`
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

type RateLimit struct {
	RequestsPerUnit uint32        `json:"requestsPerUnit"`
	Unit            RateLimitUnit `json:"unit"`
	Model           string        `json:"model"`
}

// AIExtensionStatus defines the observed state of AIExtension.
type AIExtensionStatus struct {
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// AIExtension is the Schema for the aiextensions API.
type AIExtension struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AIExtensionSpec   `json:"spec,omitempty"`
	Status AIExtensionStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// AIExtensionList contains a list of AIExtension.
type AIExtensionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AIExtension `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AIExtension{}, &AIExtensionList{})
}
