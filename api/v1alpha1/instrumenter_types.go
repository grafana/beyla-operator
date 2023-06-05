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

// Exporter type for metrics
// +kubebuilder:validation:Enum:="Prometheus";"OpenTelemetryMetrics";"OpenTelemetryTraces"
type Exporter string

const (
	ExporterPrometheus  = "Prometheus"
	ExporterOTELMetrics = "OpenTelemetryMetrics"
	ExporterOTELTraces  = "OpenTelemetryTraces"
)

// InstrumenterSpec defines the desired state of Instrumenter
type InstrumenterSpec struct {
	// Exporters define the exporter endpoints that the autoinstrumenter must support
	// +optional
	// +kubebuilder:default:={"Prometheus"}
	Export []Exporter `json:"export"`

	// Selector overrides the selection of Pods and executables to instrument
	// +kubebuilder:default:={portLabel:"grafana.com/instrument-port"}
	Selector Selector `json:"selector,omitempty"`

	// Prometheus allows configuring the autoinstrumenter as a Prometheus pull exporter.
	// +kubebuilder:default:={path:"/metrics"}
	Prometheus Prometheus `json:"prometheus"`
}

// Selector allows selecting the Pod and executable to autoinstrument
type Selector struct {
	// PortLabel specifies which Pod label would specify which executable needs to be instrumented,
	// according to the port it opens.
	// Any pod containing the label would be selected for instrumentation
	// +optional
	// +kubebuilder:default:="grafana.com/instrument-port"
	PortLabel string `json:"portLabel"`
}

type Prometheus struct {
	// +kubebuilder:default:="/metrics"
	Path string `json:"path,omitempty"`

	// +kubebuilder:default:=9102
	// +kubebuilder:validate:Minimum:=1
	// +kubebuilder:validate:Maximum:=65535
	Port int `json:"port,omitempty"`

	// +kubebuilder:default:={scrape:"prometheus.io/scrape"}
	Labels PrometheusLabels `json:"labels,omitempty"`
}

type PrometheusLabels struct {
	// +kubebuilder:default:="prometheus.io/scrape"
	Scrape string `json:"scrape,omitempty"`

	// +kubebuilder:default:="prometheus.io/scheme"
	Scheme string `json:"scheme,omitempty"`

	// +kubebuilder:default:="prometheus.io/port"
	Port string `json:"port,omitempty"`

	// +kubebuilder:default:="prometheus.io/path"
	Path string `json:"path,omitempty"`
}

// InstrumenterStatus defines the observed state of Instrumenter
type InstrumenterStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:path=instrumenters
//+kubebuilder:resource:scope=Namespaced

// Instrumenter is the Schema for the instrumenters API
type Instrumenter struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   InstrumenterSpec   `json:"spec,omitempty"`
	Status InstrumenterStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// InstrumenterList contains a list of Instrumenter
type InstrumenterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Instrumenter `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Instrumenter{}, &InstrumenterList{})
}
