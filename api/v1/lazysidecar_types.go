/*
Copyright 2022.

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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	PREFIX       = "lazy-sidecar-"
	DEFAULT_HOST = "istio-system/*"
)

// LazySidecarSpec defines the desired state of LazySidecar
type LazySidecarSpec struct {
	// WorkloadSelector is used to select the specific set of pods/VMs on which this
	// `Sidecar` configuration should be applied. If omitted, the `Sidecar`
	// configuration will be applied to all workload instances in the same namespace.
	// +optional
	WorkloadSelector map[string]string `json:"workloadSelector,omitempty"`

	// EgressHosts is used to add host to the outbound config to the sidecar;
	// if nil, the default is "istio-system/*"
	// +optional
	EgressHosts []string `json:"egressHosts,omitempty"`

	// This flag tells the controller to enable lazy sidecar mode. Defaults to true.
	// +optional
	// Enabled *bool `json:"enabled,omitempty"`

	// Middleware list are used to bind services which are registered to the system and non-automatic recognized.
	// +optional
	MiddlewareList []Middleware `json:"middlewareList,omitempty"`
}

type Middleware struct {
	ServiceName string `json:"serviceName"`
	Namespace   string `json:"namespace"`
	Port        int    `json:"port"`
	Type        string `json:"type"`
	Protocol    string `json:"protocol"`
}

// LazySidecarStatus defines the observed state of LazySidecar
type LazySidecarStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Status defines the status of the LazySidecar,contains "Succeed" and "Failed"
	Status string `json:"status"`
	// Fail message when LazySidecar occurs error
	// +optional
	FailedMsg string `json:"failedMsg,omitempty"`
	// LastUpdateTime defines last update time of the LazySidecar
	LastUpdateTimestamp metav1.Time `json:"lastUpdateTimestamp,omitempty"`
	// Upstream defines the workload's upstream service
	// Upstream []istiov1beta1.IstioEgressListener `json:"upstream,omitempty"`
	// SidecarName defines the Sidecar name which is derived from LaySidecar
	// +optional
	SidecarName string `json:"sidecarName,omitempty"`
	// EnvoyFilterName defines the EnvoyFilter name which is derived from LazySidecar
	// +optional
	EnvoyFilterName string `json:"envoyFilterName,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// LazySidecar is the Schema for the lazysidecars API
type LazySidecar struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LazySidecarSpec   `json:"spec,omitempty"`
	Status LazySidecarStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// LazySidecarList contains a list of LazySidecar
type LazySidecarList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []LazySidecar `json:"items"`
}

func init() {
	SchemeBuilder.Register(&LazySidecar{}, &LazySidecarList{})
}
