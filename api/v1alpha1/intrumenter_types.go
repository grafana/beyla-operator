/*
Copyright 2023 Grafana Labs <hello@grafana.com>.

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

const (
	labelOpenPort = "autoinstrument.open.port"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// IntrumenterSpec defines the desired state of Intrumenter
type IntrumenterSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Selector overrides the selection of Pods and executables to instrument
	// +optional
	Selector Selector `json:"selector"`
}

// Selector allows selecting the Pod and executable to autoinstrument
type Selector struct {
	// PortLabel specifies which Pod label would specify which executable needs to be instrumented,
	// according to the port it opens.
	// Any pod containing the label would be selected for instrumentation
	// +kubebuilder:default:="autoinstrument.open.port"
	PortLabel string `json:"portLabel"`
}

// IntrumenterStatus defines the observed state of Intrumenter
type IntrumenterStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:path=instrumenters
//+kubebuilder:resource:scope=Namespaced

// Intrumenter is the Schema for the instrumenters API
type Intrumenter struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IntrumenterSpec   `json:"spec,omitempty"`
	Status IntrumenterStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// IntrumenterList contains a list of Intrumenter
type IntrumenterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Intrumenter `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Intrumenter{}, &IntrumenterList{})
}
