package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MeshConfig is the configuration for the service mesh overall
// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type MeshConfig struct {
	metav1.TypeMeta   `json:",inline" yaml:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" yaml:"metadata,omitempty"`

	Spec MeshConfigSpec `json:"spec,omitempty" yaml:"spec,omitempty"`
}

// MeshConfigSpec is the spec for OSM's configuration
type MeshConfigSpec struct {
	Sidecar       SidecarSpec       `json:"sidecar,omitempty"`
	Traffic       TrafficSpec       `json:"traffic,omitempty"`
	Observability ObservabilitySpec `json:"observability,omitempty"`
	Certificate   CertificateSpec   `json:"certificate,omitempty"`
}

// SidecarSpec is the spec for OSM's sidecar configuration
type SidecarSpec struct {
	EnablePrivilegedInitContainer bool                        `json:"enablePrivilegedInitContainer,omitempty"`
	LogLevel                      string                      `json:"logLevel,omitempty"`
	EnvoyImage                    string                      `json:"envoyImage,omitempty"`
	InitContainerImage            string                      `json:"initContainerImage,omitempty"`
	MaxDataPlaneConnections       int                         `json:"maxDataPlaneConnections,omitempty"`
	ConfigResyncInterval          string                      `json:"configResyncInterval,omitempty"`
	Resources                     corev1.ResourceRequirements `json:"resources,omitempty"`
}

// TrafficSpec is the spec for OSM's traffic management configuration
type TrafficSpec struct {
	EnableEgress                      bool     `json:"enableEgress,omitempty"`
	OutboundIPRangeExclusionList      []string `json:"outboundIPRangeExclusionList,omitempty"`
	OutboundPortExclusionList         []int    `json:"outboundPortExclusionList,omitempty"`
	UseHTTPSIngress                   bool     `json:"useHTTPSIngress,omitempty"`
	EnablePermissiveTrafficPolicyMode bool     `json:"enablePermissiveTrafficPolicyMode,omitempty"`
}

// ObservabilitySpec is the spec for OSM's observability related configuration
type ObservabilitySpec struct {
	EnableDebugServer  bool        `json:"enableDebugServer,omitempty"`
	PrometheusScraping bool        `json:"prometheusScraping,omitempty"`
	Tracing            TracingSpec `json:"tracing,omitempty"`
}

// TracingSpec is the spec for OSM's tracing configuration
type TracingSpec struct {
	Enable   bool   `json:"enable,omitempty"`
	Port     int16  `json:"port,omitempty"`
	Address  string `json:"address,omitempty"`
	Endpoint string `json:"endpoint,omitempty"`
}

// CertificateSpec is the spec for OSM's certificate management configuration
type CertificateSpec struct {
	ServiceCertValidityDuration string `json:"serviceCertValidityDuration,omitempty"`
}

// MeshConfigList lists the MeshConfig objects
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type MeshConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []MeshConfig `json:"items"`
}
