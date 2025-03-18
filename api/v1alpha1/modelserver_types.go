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

// ModelServerSpec defines the desired state of ModelServer.
type ModelServerSpec struct {
	// The backend model that is actually accessed. For access to the model adapter,
	// it should be the base model where the model adapter is located. For other scenarios,
	// if model name is different from this field, it should be overwritten by this field.
	Model string `json:"model"` // Rename to `Base model`?
	// The inference engine used to manage the model.
	InferenceEngine string `json:"inferenceEngine"`
	// Used to select pods where the model is located.
	WorkloadSelector *WorkloadSelector `json:"workloadSelector"`
	// Traffic Policy for accessing the model.
	TrafficPolicy *TrafficPolicy `json:"trafficPolicy,omitempty"`
}

type WorkloadSelector struct {
	MatchLabels map[string]string `json:"matchLabels,omitempty"`
}

type TrafficPolicy struct {
	// LoadBalancer *LoadBalancerSettings `json:"loadBalancer,omitempty"`
	Timeout *metav1.Duration `json:"timeout,omitempty"`
	Retry   *Retry           `json:"retry,omitempty"`
}

type Retry struct {
	Attempts int32 `json:"attempts"`
}

// ModelServerStatus defines the observed state of ModelServer.
type ModelServerStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// ModelServer is the Schema for the modelservers API.
type ModelServer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ModelServerSpec   `json:"spec,omitempty"`
	Status ModelServerStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ModelServerList contains a list of ModelServer.
type ModelServerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ModelServer `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ModelServer{}, &ModelServerList{})
}
