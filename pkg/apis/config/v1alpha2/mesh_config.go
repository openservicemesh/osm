package v1alpha2

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

// LocalProxyMode is a type alias representing the way the envoy sidecar proxies to the main application
type LocalProxyMode string

const (
	// LocalProxyModeLocalhost indicates the the sidecar should communicate with the main application over localhost
	LocalProxyModeLocalhost LocalProxyMode = "Localhost"
	// LocalProxyModePodIP indicates that the sidecar should communicate with the main application via the pod ip
	LocalProxyModePodIP LocalProxyMode = "PodIP"
)

// SidecarSpec is the type used to represent the specifications for the proxy sidecar.
type SidecarSpec struct {
	// EnablePrivilegedInitContainer defines a boolean indicating whether the init container for a meshed pod should run as privileged.
	EnablePrivilegedInitContainer bool `json:"enablePrivilegedInitContainer"`

	// LogLevel defines the logging level for the sidecar's logs. Non developers should generally never set this value. In production environments the LogLevel should be set to error.
	LogLevel string `json:"logLevel,omitempty"`

	// EnvoyImage defines the container image used for the Envoy proxy sidecar.
	EnvoyImage string `json:"envoyImage,omitempty"`

	// EnvoyWindowsImage defines the windows container image used for the Envoy proxy sidecar.
	EnvoyWindowsImage string `json:"envoyWindowsImage,omitempty"`

	// InitContainerImage defines the container image used for the init container injected to meshed pods.
	InitContainerImage string `json:"initContainerImage,omitempty"`

	// MaxDataPlaneConnections defines the maximum allowed data plane connections from a proxy sidecar to the OSM controller.
	MaxDataPlaneConnections int `json:"maxDataPlaneConnections,omitempty"`

	// ConfigResyncInterval defines the resync interval for regular proxy broadcast updates.
	ConfigResyncInterval string `json:"configResyncInterval,omitempty"`

	// Resources defines the compute resources for the sidecar.
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	// TLSMinProtocolVersion defines the minimum TLS protocol version that the sidecar supports. Valid TLS protocol versions are TLS_AUTO, TLSv1_0, TLSv1_1, TLSv1_2 and TLSv1_3.
	TLSMinProtocolVersion string `json:"tlsMinProtocolVersion,omitempty"`

	// TLSMaxProtocolVersion defines the maximum TLS protocol version that the sidecar supports. Valid TLS protocol versions are TLS_AUTO, TLSv1_0, TLSv1_1, TLSv1_2 and TLSv1_3.
	TLSMaxProtocolVersion string `json:"tlsMaxProtocolVersion,omitempty"`

	// CipherSuites defines a list of ciphers that listener supports when negotiating TLS 1.0-1.2. This setting has no effect when negotiating TLS 1.3. For valid cipher names, see the latest OpenSSL ciphers manual page. E.g. https://www.openssl.org/docs/man1.1.1/apps/ciphers.html.
	CipherSuites []string `json:"cipherSuites,omitempty"`

	// ECDHCurves defines a list of ECDH curves that TLS connection supports. If not specified, the curves are [X25519, P-256] for non-FIPS build and P-256 for builds using BoringSSL FIPS.
	ECDHCurves []string `json:"ecdhCurves,omitempty"`

	// LocalProxyMode defines the network interface the envoy proxy will use to send traffic to the backend service application. Acceptable values are [`Localhost`, `PodIP`]. The default is `Localhost`
	LocalProxyMode LocalProxyMode `json:"localProxyMode,omitempty"`
}

// TrafficSpec is the type used to represent OSM's traffic management configuration.
type TrafficSpec struct {
	// EnableEgress defines a boolean indicating if mesh-wide Egress is enabled.
	EnableEgress bool `json:"enableEgress"`

	// OutboundIPRangeExclusionList defines a global list of IP address ranges to exclude from outbound traffic interception by the sidecar proxy.
	OutboundIPRangeExclusionList []string `json:"outboundIPRangeExclusionList"`

	// OutboundIPRangeInclusionList defines a global list of IP address ranges to include for outbound traffic interception by the sidecar proxy.
	// IP addresses outside this range will be excluded from outbound traffic interception by the sidecar proxy.
	OutboundIPRangeInclusionList []string `json:"outboundIPRangeInclusionList"`

	// OutboundPortExclusionList defines a global list of ports to exclude from outbound traffic interception by the sidecar proxy.
	OutboundPortExclusionList []int `json:"outboundPortExclusionList"`

	// InboundPortExclusionList defines a global list of ports to exclude from inbound traffic interception by the sidecar proxy.
	InboundPortExclusionList []int `json:"inboundPortExclusionList"`

	// EnablePermissiveTrafficPolicyMode defines a boolean indicating if permissive traffic policy mode is enabled mesh-wide.
	EnablePermissiveTrafficPolicyMode bool `json:"enablePermissiveTrafficPolicyMode"`

	// InboundExternalAuthorization defines a ruleset that, if enabled, will configure a remote external authorization endpoint
	// for all inbound and ingress traffic in the mesh.
	InboundExternalAuthorization ExternalAuthzSpec `json:"inboundExternalAuthorization,omitempty"`

	// NetworkInterfaceExclusionList defines a global list of network interface
	// names to exclude from inbound and outbound traffic interception by the
	// sidecar proxy.
	NetworkInterfaceExclusionList []string `json:"networkInterfaceExclusionList"`
}

// ObservabilitySpec is the type to represent OSM's observability configurations.
type ObservabilitySpec struct {
	// OSMLogLevel defines the log level for OSM control plane logs.
	OSMLogLevel string `json:"osmLogLevel,omitempty"`

	// EnableDebugServer defines if the debug endpoint on the OSM controller pod is enabled.
	EnableDebugServer bool `json:"enableDebugServer"`

	// Tracing defines OSM's tracing configuration.
	Tracing TracingSpec `json:"tracing,omitempty"`
}

// TracingSpec is the type to represent OSM's tracing configuration.
type TracingSpec struct {
	// Enable defines a boolean indicating if the sidecars are enabled for tracing.
	Enable bool `json:"enable"`

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
	Enable bool `json:"enable"`

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
	FailureModeAllow bool `json:"failureModeAllow"`
}

// CertificateSpec is the type to reperesent OSM's certificate management configuration.
type CertificateSpec struct {
	// ServiceCertValidityDuration defines the service certificate validity duration.
	ServiceCertValidityDuration string `json:"serviceCertValidityDuration,omitempty"`

	// CertKeyBitSize defines the certicate key bit size.
	CertKeyBitSize int `json:"certKeyBitSize,omitempty"`

	// IngressGateway defines the certificate specification for an ingress gateway.
	// +optional
	IngressGateway *IngressGatewayCertSpec `json:"ingressGateway,omitempty"`
}

// IngressGatewayCertSpec is the type to represent the certificate specification for an ingress gateway.
type IngressGatewayCertSpec struct {
	// SubjectAltNames defines the Subject Alternative Names (domain names and IP addresses) secured by the certificate.
	SubjectAltNames []string `json:"subjectAltNames"`

	// ValidityDuration defines the validity duration of the certificate.
	ValidityDuration string `json:"validityDuration"`

	// Secret defines the secret in which the certificate is stored.
	Secret corev1.SecretReference `json:"secret"`
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
	EnableWASMStats bool `json:"enableWASMStats"`

	// EnableEgressPolicy defines if OSM's Egress policy is enabled.
	EnableEgressPolicy bool `json:"enableEgressPolicy"`

	// EnableSnapshotCacheMode defines if XDS server starts with snapshot cache.
	EnableSnapshotCacheMode bool `json:"enableSnapshotCacheMode"`

	//EnableAsyncProxyServiceMapping defines if OSM will map proxies to services asynchronously.
	EnableAsyncProxyServiceMapping bool `json:"enableAsyncProxyServiceMapping"`

	// EnableIngressBackendPolicy defines if OSM will use the IngressBackend API to allow ingress traffic to
	// service mesh backends.
	EnableIngressBackendPolicy bool `json:"enableIngressBackendPolicy"`

	// EnableEnvoyActiveHealthChecks defines if OSM will Envoy active health
	// checks between services allowed to communicate.
	EnableEnvoyActiveHealthChecks bool `json:"enableEnvoyActiveHealthChecks"`

	// EnableRetryPolicy defines if retry policy is enabled.
	EnableRetryPolicy bool `json:"enableRetryPolicy"`
}
