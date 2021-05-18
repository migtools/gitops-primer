/*
Copyright 2021.

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

// PrimerSpec defines the desired state of Primer
type PrimerSpec struct {
	Branch string `json:"branch"`
	Repo string `json:"repo"`
	Action string `json:"action"`
}

// PrimerStatus defines the observed state of Primer
type PrimerStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Primer is the Schema for the primers API
type Primer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PrimerSpec   `json:"spec,omitempty"`
	Status PrimerStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// PrimerList contains a list of Primer
type PrimerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Primer `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Primer{}, &PrimerList{})
}
