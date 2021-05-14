package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MeshConfig is the type used to represent the mesh configuration.
// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type MeshConfig struct {
	// Object's type metadata.
	metav1.TypeMeta `json:",inline" yaml:",inline"`

	// Object's metadata.
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty" yaml:"metadata,omitempty"`

	// Spec is the MeshConfig specification.
	// +optional
	Spec MeshConfigSpec `json:"spec,omitempty" yaml:"spec,omitempty"`
}

// MeshConfigSpec is the spec for OSM's configuration.
type MeshConfigSpec struct {
	// Sidecar defines the configurations of the proxy sidecar in a mesh.
	Sidecar SidecarSpec `json:"sidecar,omitempty"`

	// Traffic defines the traffic management configurations for a mesh instance.
	Traffic TrafficSpec `json:"traffic,omitempty"`

	// Observalility defines the observability configurations for a mesh instance.
	Observability ObservabilitySpec `json:"observability,omitempty"`

	// Certificate defines the certificate management configurations for a mesh instance.
	Certificate CertificateSpec `json:"certificate,omitempty"`
}

// SidecarSpec is the type used to represent the specifications for the proxy sidecar.
type SidecarSpec struct {
	// EnablePrivilegedInitContainer defines a boolean indicating whether the init container for a meshed pod should run as privileged.
	EnablePrivilegedInitContainer bool `json:"enablePrivilegedInitContainer,omitempty"`

	// LogLevel defines the  logging level for the sidecar's logs.
	LogLevel string `json:"logLevel,omitempty"`

	// EnvoyImage defines the container image used for the Envoy proxy sidecar.
	EnvoyImage string `json:"envoyImage,omitempty"`

	// InitContainerImage defines the container image used for the init container injected to meshed pods.
	InitContainerImage string `json:"initContainerImage,omitempty"`

	// MaxDataPlaneConnections defines the maximum allowed data plane connections from a proxy sidecar to the OSM controller.
	MaxDataPlaneConnections int `json:"maxDataPlaneConnections,omitempty"`

	// ConfigResyncInterval defines the resync interval for regular proxy broadcast updates.
	ConfigResyncInterval string `json:"configResyncInterval,omitempty"`

	// Resources defines the compute resources for the sidecar.
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`
}

// TrafficSpec is the type used to represent OSM's traffic management configuration.
type TrafficSpec struct {
	// EnableEgress defines a boolean indicating if mesh-wide Egress is enabled.
	EnableEgress bool `json:"enableEgress,omitempty"`

	// OutboundIPRangeExclusionList defines a global list of IP address ranges to exclude from outbound traffic interception by the sidecar proxy.
	OutboundIPRangeExclusionList []string `json:"outboundIPRangeExclusionList,omitempty"`

	// OutboundPortExclusionList defines a global list of ports to exclude from outbound traffic interception by the sidecar proxy.
	OutboundPortExclusionList []int `json:"outboundPortExclusionList,omitempty"`

	// UseHTTPSIngress defines a boolean indicating if HTTPS Ingress is enabled globally in the mesh.
	UseHTTPSIngress bool `json:"useHTTPSIngress,omitempty"`

	// EnablePermissiveTrafficPolicyMode defines a boolean indicating if permissive traffic policy mode is enabled mesh-wide.
	EnablePermissiveTrafficPolicyMode bool `json:"enablePermissiveTrafficPolicyMode,omitempty"`
}

// ObservabilitySpec is the type to represent OSM's observability configurations.
type ObservabilitySpec struct {
	// EnableDebugServer defines if the debug endpoint on the OSM controller pod is enabled.
	EnableDebugServer bool `json:"enableDebugServer,omitempty"`

	// PrometheusScraping defines a boolean indicating if sidecars should be configured for Prometheus metrics scraping.
	PrometheusScraping bool `json:"prometheusScraping,omitempty"`

	// Tracing defines OSM's tracing configuration.
	Tracing TracingSpec `json:"tracing,omitempty"`
}

// TracingSpec is the type to represent OSM's tracing configuration.
type TracingSpec struct {
	// Enable defines a boolean indicating if the sidecars are enabled for tracing.
	Enable bool `json:"enable,omitempty"`

	// Port defines the tracing collector's port.
	Port int16 `json:"port,omitempty"`

	// Address defines the tracing collectio's hostname.
	Address string `json:"address,omitempty"`

	// Endpoint defines the API endpoint for tracing requests sent to the collector.
	Endpoint string `json:"endpoint,omitempty"`
}

// CertificateSpec is type to reperesent OSM's certificate management configuration.
type CertificateSpec struct {
	// ServiceCertValidityDuration defines the service certificate validity duration.
	ServiceCertValidityDuration string `json:"serviceCertValidityDuration,omitempty"`
}

// MeshConfigList lists the MeshConfig objects.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type MeshConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []MeshConfig `json:"items"`
}
