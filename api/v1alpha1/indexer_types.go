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
	"github.com/rss3-network/node/config"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// IndexerSpec defines the desired state of Indexer
type IndexerSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Network  string `json:"network,omitempty"`
	Worker   string `json:"worker,omitempty"`
	Endpoint string `json:"endpoint,omitempty"`

	Params      IndexerParams `json:"params,omitempty"`
	DatabaseRef DatabaseRef   `json:"database"`
}

// IndexerStatus defines the observed state of Indexer
type IndexerStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Status UpdateStatus `json:"status,omitempty"`
	// Conditions store the status conditions of the Indexer instances
	// +operator-sdk:csv:customresourcedefinitions:type=status
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Indexer is the Schema for the indexers API
type Indexer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IndexerSpec   `json:"spec,omitempty"`
	Status IndexerStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// IndexerList contains a list of Indexer
type IndexerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Indexer `json:"items"`
}

func (cr *Indexer) SpecDiff() bool {
	var preSpec IndexerSpec
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

func (cr *Indexer) PatchApplyAnnotations() (client.Patch, error) {
	data, err := json.Marshal(cr.Spec)
	if err != nil {
		return nil, fmt.Errorf("cannot marshal spec: %w", err)
	}
	patch := fmt.Sprintf(`{"metadata":{"annotations":{"%s":"%s"}}}`, lastAppliedConfigAnnotation, string(data))
	return client.RawPatch(types.MergePatchType, []byte(patch)), nil
}

func (cr *Indexer) SetUpdateStatusTo(ctx context.Context, r client.Client, status UpdateStatus, maybeReason error) error {
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
				Type:    "Ready",
				Status:  metav1.ConditionTrue,
				Reason:  "Ready",
				Message: "Hub is ready",
			},
		}
	}

	if err := r.Status().Update(ctx, cr); err != nil {
		return fmt.Errorf("cannot update status for indexer: %s: %w", cr.Name, err)
	}
	return nil
}

func (cr *Indexer) SelectorLabels() map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":      "node",
		"app.kubernetes.io/instance":  cr.Name,
		"app.kubernetes.io/component": "indexer",

		"app.kubernetes.io/part-of":    "node-operator",
		"app.kubernetes.io/managed-by": "node-operator",
		"app.kubernetes.io/created-by": "controller-manager",
	}
}

func (cr *Indexer) PodLabels() map[string]string {
	selectorLabels := cr.SelectorLabels()
	return labels.Merge(selectorLabels, nil)
}

type IndexerParams map[string]runtime.RawExtension

func (cr IndexerParams) Options() *config.Options {
	option := config.Options{}

	for k, v := range cr {
		option[k] = v.String()
	}
	return &option
}

func init() {
	SchemeBuilder.Register(&Indexer{}, &IndexerList{})
}
