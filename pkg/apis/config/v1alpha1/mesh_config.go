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

	// FeatureFlags defines the feature flags for a mesh instance.
	FeatureFlags FeatureFlags `json:"featureFlags,omitempty"`
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

	// InboundPortExclusionList defines a global list of ports to exclude from inbound traffic interception by the sidecar proxy.
	InboundPortExclusionList []int `json:"inboundPortExclusionList,omitempty"`

	// UseHTTPSIngress defines a boolean indicating if HTTPS Ingress is enabled globally in the mesh.
	UseHTTPSIngress bool `json:"useHTTPSIngress,omitempty"`

	// EnablePermissiveTrafficPolicyMode defines a boolean indicating if permissive traffic policy mode is enabled mesh-wide.
	EnablePermissiveTrafficPolicyMode bool `json:"enablePermissiveTrafficPolicyMode,omitempty"`

	// InboundExternalAuthorization defines a ruleset that, if enabled, will configure a remote external authorization endpoint
	// for all inbound and ingress traffic in the mesh.
	InboundExternalAuthorization ExternalAuthzSpec `json:"inboundExternalAuthorization,omitempty"`
}

// ObservabilitySpec is the type to represent OSM's observability configurations.
type ObservabilitySpec struct {
	// EnableDebugServer defines if the debug endpoint on the OSM controller pod is enabled.
	EnableDebugServer bool `json:"enableDebugServer,omitempty"`

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

// ExternalAuthzSpec is a type to represent external authorization configuration.
type ExternalAuthzSpec struct {
	// Enable defines a boolean indicating if the external authorization policy is to be enabled.
	Enable bool `json:"enable,omitempty"`

	// Address defines the remote address of the external authorization endpoint.
	Address string `json:"address,omitempty"`

	// Port defines the destination port of the remote external authorization endpoint.
	Port uint16 `json:"port,omitempty"`

	// StatPrefix defines a prefix for the stats sink for this external authorization policy.
	StatPrefix string `json:"statPrefix,omitempty"`

	// Timeout defines the timeout in which a response from the external authorization endpoint.
	// is expected to execute.
	Timeout string `json:"timeout,omitempty"`

	// FailureModeAllow defines a boolean indicating if traffic should be allowed on a failure to get a
	// response against the external authorization endpoint.
	FailureModeAllow bool `json:"failureModeAllow,omitempty"`
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

// FeatureFlags is a type to represent OSM's feature flags.
type FeatureFlags struct {
	// EnableWASMStats defines if WASM Stats are enabled.
	EnableWASMStats bool `json:"enableWASMStats,omitempty"`

	// EnableEgressPolicy defines if OSM's Egress policy is enabled.
	EnableEgressPolicy bool `json:"enableEgressPolicy,omitempty"`

	// EnableMulticlusterMode defines if Multicluster mode is enabled.
	EnableMulticlusterMode bool `json:"enableMulticlusterMode,omitempty"`

	// EnableSnapshotCacheMode defines if XDS server starts with snapshot cache.
	EnableSnapshotCacheMode bool `json:"enableSnapshotCacheMode,omitempty"`

	// EnableOSMGateway defines if OSM gateway is enabled.
	EnableOSMGateway bool `json:"enableOSMGateway,omitempty"`

	//EnableAsyncProxyServiceMapping defines if OSM will map proxies to services asynchronously.
	EnableAsyncProxyServiceMapping bool `json:"enableAsyncProxyServiceMapping,omitempty"`
}
