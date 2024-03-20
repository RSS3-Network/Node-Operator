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
	"k8s.io/apimachinery/pkg/util/json"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// IndexerSpec defines the desired state of Indexer
type IndexerSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of Indexer. Edit indexer_types.go to remove/update

	// Network string must be one of string slice var NetworkEnum
	// +kubebuilder:validation:Enum=mainnet;testnet;rinkeby;goerli;ropsten;localhost
	Network     string        `json:"network,omitempty"`
	Worker      string        `json:"worker,omitempty"`
	Params      IndexerParams `json:"params,omitempty"`
	DatabaseRef DatabaseRef   `json:"database"`
}

// IndexerStatus defines the observed state of Indexer
type IndexerStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

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

type IndexerParams map[string]string

func (ip *IndexerParams) String() string {
	b, err := json.Marshal(ip)
	if err != nil {
		return "{}"
	}
	return string(b)
}

func init() {
	SchemeBuilder.Register(&Indexer{}, &IndexerList{})
}
