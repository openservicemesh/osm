package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

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
	Sidecar       SidecarSpec       `json:"sidecar,omitempty" yaml:"sidecar,omitempty"`
	Traffic       TrafficSpec       `json:"traffic,omitempty" yaml:"traffic,omitempty"`
	Observability ObservabilitySpec `json:"observability,omitempty" yaml:"observability,omitempty"`
	Certificate   CertificateSpec   `json:"certificate,omitempty" yaml:"certificate,omitempty"`
}

// SidecarSpec is the spec for OSM's sidecar configuration
type SidecarSpec struct {
	EnablePrivilegedInitContainer bool   `json:"enablePrivilegedInitContainer,omitempty" yaml:"enablePrivilegedInitContainer,omitempty"`
	LogLevel                      string `json:"logLevel,omitempty" yaml:"logLevel,omitempty"`
	MaxDataPlaneConnections       int    `json:"maxMaxPlaneConnections,omitempty" yaml:"max_data_plane_connections,omitempty"`
	ConfigResyncInterval          string `json:"configResyncInterval,omitempty" yaml:"config_resync_interval,omitempty"`
}

// TrafficSpec is the spec for OSM's traffic management configuration
type TrafficSpec struct {
	EnableEgress                      bool     `json:"enableEgress,omitempty" yaml:"enableEgress,omitempty"`
	OutboundIPRangeExclusionList      []string `json:"outboundIPRangeExclusionList,omitempty" yaml:"outboundIPRangeExclusionList,omitempty"`
	UseHTTPSIngress                   bool     `json:"useHTTPSIngress,omitempty" yaml:"useHTTPSIngress,omitempty"`
	EnablePermissiveTrafficPolicyMode bool     `json:"enablePermissiveTrafficPolicyMode,omitempty" yaml:"enablePermissiveTrafficPolicyMode,omitempty"`
}

// ObservabilitySpec is the spec for OSM's observability related configuration
type ObservabilitySpec struct {
	EnableDebugServer  bool        `json:"enableDebugServer,omitempty" yaml:"enableDebugServer,omitempty"`
	PrometheusScraping bool        `json:"prometheusScraping,omitempty" yaml:"prometheusScraping,omitempty"`
	Tracing            TracingSpec `json:"tracing,omitempty" yaml:"tracing,omitempty"`
}

// TracingSpec is the spec for OSM's tracing configuration
type TracingSpec struct {
	Enable   bool   `json:"enable,omitempty" yaml:"enable,omitempty"`
	Port     int16  `json:"port,omitempty" yaml:"port,omitempty"`
	Address  string `json:"address,omitempty" yaml:"address,omitempty"`
	Endpoint string `json:"endpoint,omitempty" yaml:"endpoint,omitempty"`
}

// CertificateSpec is the spec for OSM's certificate management configuration
type CertificateSpec struct {
	ServiceCertValidityDuration string `json:"serviceCertValidityDuration,omitempty" yaml:"serviceCertValidityDuration,omitempty"`
}

// MeshConfigList lists the MeshConfig objects
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type MeshConfigList struct {
	metav1.TypeMeta `json:",inline" yaml:",inline"`
	metav1.ListMeta `json:"metadata,omitempty" yaml:"metadata,omitempty"`

	Items []MeshConfig `json:"items" yaml:"items"`
}
