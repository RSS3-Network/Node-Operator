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
	"fmt"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// HubSpec defines the desired state of Hub
type HubSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Replicas    int32       `json:"replicas,omitempty"`
	DatabaseRef DatabaseRef `json:"database"`
}

// HubStatus defines the observed state of Hub
type HubStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Conditions store the status conditions of the Hub instances
	// +operator-sdk:csv:customresourcedefinitions:type=status
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Hub is the Schema for the hubs API
type Hub struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   HubSpec   `json:"spec,omitempty"`
	Status HubStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// HubList contains a list of Hub
type HubList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Hub `json:"items"`
}

type DatabaseRef struct {
	Driver    string               `json:"driver"`
	Partition bool                 `json:"partition"`
	UriRef    *corev1.EnvVarSource `json:"uri"`
}

func (d *DatabaseRef) EnvVars() []corev1.EnvVar {
	return []corev1.EnvVar{
		{Name: "DATABASE_DRIVER", Value: d.Driver},
		{Name: "DATABASE_PARITION", Value: fmt.Sprintf("%t", d.Partition)},
		{Name: "DATABASE_URI", ValueFrom: d.UriRef},
	}
}

func init() {
	SchemeBuilder.Register(&Hub{}, &HubList{})
}
