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
	"bytes"
	"context"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// HubSpec defines the desired state of Hub
type HubSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Replicas    *int32      `json:"replicas,omitempty"`
	DatabaseRef DatabaseRef `json:"database"`
}

// HubStatus defines the observed state of Hub
type HubStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Status UpdateStatus `json:"status,omitempty"`
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
		{Name: "NODE_DATABASE_DRIVER", Value: d.Driver},
		{Name: "NODE_DATABASE_PARITION", Value: fmt.Sprintf("%t", d.Partition)},
		{Name: "NODE_DATABASE_URI", ValueFrom: d.UriRef},
	}
}

func (cr *Hub) SpecDiff() bool {
	var preSpec HubSpec
	lastAppliedConfig := cr.GetAnnotations()[lastAppliedConfigAnnotation]
	if len(lastAppliedConfig) == 0 {
		return true
	}
	if err := json.Unmarshal([]byte(lastAppliedConfig), &preSpec); err != nil {
		return true
	}
	specData, _ := json.Marshal(cr.Spec)
	return !bytes.Equal(specData, []byte(lastAppliedConfig))
}

func (cr *Hub) PatchApplyAnnotations() (client.Patch, error) {
	data, err := json.Marshal(cr.Spec)
	if err != nil {
		return nil, fmt.Errorf("cannot marshal spec: %w", err)
	}
	patch := fmt.Sprintf(`{"metadata":{"annotations":{"%s": %q}}}`, lastAppliedConfigAnnotation, string(data))
	return client.RawPatch(types.MergePatchType, []byte(patch)), nil
}

func (cr *Hub) SetUpdateStatusTo(ctx context.Context, r client.Client, status UpdateStatus, maybeReason error) error {
	cr.Status.Status = status

	switch status {
	case UpdateStatusFailed:
	case UpdateStatusDegraded:
	case UpdateStatusExpanding:
		if maybeReason != nil {
			cr.Status.Conditions = []metav1.Condition{
				{
					Type:               "Expanding",
					Status:             metav1.ConditionFalse,
					Reason:             "Expanding",
					Message:            maybeReason.Error(),
					LastTransitionTime: metav1.Now(),
				},
			}
		}
	default:
		cr.Status.Conditions = []metav1.Condition{
			{
				Type:               "Ready",
				Status:             metav1.ConditionTrue,
				Reason:             "Ready",
				Message:            "Hub is ready",
				LastTransitionTime: metav1.Now(),
			},
		}
	}

	if err := r.Status().Update(ctx, cr); err != nil {
		return fmt.Errorf("cannot update status for hub: %s: %w", cr.Name, err)
	}
	return nil
}

func (cr *Hub) SelectorLabels() map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":      "node",
		"app.kubernetes.io/instance":  cr.Name,
		"app.kubernetes.io/component": "hub",

		"app.kubernetes.io/part-of":    "node-operator",
		"app.kubernetes.io/managed-by": "node-operator",
		"app.kubernetes.io/created-by": "controller-manager",
	}
}

func (cr *Hub) PodLabels() map[string]string {
	selectorLabels := cr.SelectorLabels()
	return labels.Merge(selectorLabels, nil)
}

func init() {
	SchemeBuilder.Register(&Hub{}, &HubList{})
}
