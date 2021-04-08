package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// MeshConfig is the configuration for the service mesh overall
// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type MeshConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec MeshConfigSpec `json:"spec,omitempty"`
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
	EnablePrivilegedInitContainer bool   `json:"enable_privileged_init_container,omitempty"`
	LogLevel                      string `json:"log_level,omitempty" default:"error"`
}

// TrafficSpec is the spec for OSM's traffic management configuration
type TrafficSpec struct {
	Egress                            bool     `json:"egress,omitempty"`
	OutboundIPRangeExclusionList      []string `json:"outbound_ip_range_exclusion_list,omitempty"`
	UseHTTPSIngress                   bool     `json:"use_https_ingress,omitempty"`
	EnablePermissiveTrafficPolicyMode bool     `json:"enable_permissive_traffic_policy_mode,omitempty"`
}

// ObservabilitySpec is the spec for OSM's observability related configuration
type ObservabilitySpec struct {
	EnableDebugServer bool        `json:"enable_debug_server,omitempty" default:"true"`
	Tracing           TracingSpec `json:"tracing,omitempty"`
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
	ServiceCertValidityDuration string `json:"service_cert_validity_duration,omitempty" default:"24h"`
}

// MeshConfigList lists the MeshConfig objects
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type MeshConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []MeshConfig `json:"items"`
}
