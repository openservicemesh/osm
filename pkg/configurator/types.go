// Package configurator implements the Configurator interface that provides APIs to retrieve OSM control plane configurations.
package configurator

import (
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"

	configv1alpha1 "github.com/openservicemesh/osm/pkg/apis/config/v1alpha1"

	"github.com/openservicemesh/osm/pkg/auth"
	"github.com/openservicemesh/osm/pkg/logger"
)

var (
	log = logger.New("configurator")
)

// client is the type used to represent the Kubernetes client for the config.openservicemesh.io API group
type client struct {
	osmNamespace   string
	informer       cache.SharedIndexInformer
	cache          cache.Store
	meshConfigName string
}

// Configurator is the controller interface for K8s namespaces
type Configurator interface {
	// GetMeshConfig returns the MeshConfig resource corresponding to the control plane
	GetMeshConfig() *configv1alpha1.MeshConfig

	// GetOSMNamespace returns the namespace in which OSM controller pod resides
	GetOSMNamespace() string

	// GetMeshConfigJSON returns the MeshConfig in pretty JSON (human readable)
	GetMeshConfigJSON() (string, error)

	// IsPermissiveTrafficPolicyMode determines whether we are in "allow-all" mode or SMI policy (block by default) mode
	IsPermissiveTrafficPolicyMode() bool

	// IsEgressEnabled determines whether egress is globally enabled in the mesh or not
	IsEgressEnabled() bool

	// IsDebugServerEnabled determines whether osm debug HTTP server is enabled
	IsDebugServerEnabled() bool

	// IsTracingEnabled returns whether tracing is enabled
	IsTracingEnabled() bool

	// GetTracingHost is the host to which we send tracing spans
	GetTracingHost() string

	// GetTracingPort returns the tracing listener port
	GetTracingPort() uint32

	// GetTracingEndpoint returns the collector endpoint
	GetTracingEndpoint() string

	// GetMaxDataPlaneConnections returns the max data plane connections allowed, 0 if disabled
	GetMaxDataPlaneConnections() int

	// GetOsmLogLevel returns the configured OSM log level
	GetOSMLogLevel() string

	// GetEnvoyLogLevel returns the envoy log level
	GetEnvoyLogLevel() string

	// GetEnvoyImage returns the envoy image
	GetEnvoyImage() string

	// GetEnvoyWindowsImage returns the envoy windows image
	GetEnvoyWindowsImage() string

	// GetInitContainerImage returns the init container image
	GetInitContainerImage() string

	// GetServiceCertValidityPeriod returns the validity duration for service certificates
	GetServiceCertValidityPeriod() time.Duration

	// GetCertKeyBitSize returns the certificate key bit size
	GetCertKeyBitSize() int

	// GetOutboundIPRangeExclusionList returns the list of IP ranges of the form x.x.x.x/y to exclude from outbound sidecar interception
	GetOutboundIPRangeExclusionList() []string

	// GetOutboundPortExclusionList returns the list of ports to exclude from outbound sidecar interception
	GetOutboundPortExclusionList() []int

	// GetInboundPortExclusionList returns the list of ports to exclude from inbound sidecar interception
	GetInboundPortExclusionList() []int

	// IsPrivilegedInitContainer determines whether init containers should be privileged
	IsPrivilegedInitContainer() bool

	// GetConfigResyncInterval returns the duration for resync interval.
	// If error or non-parsable value, returns 0 duration
	GetConfigResyncInterval() time.Duration

	// GetProxyResources returns the `Resources` configured for proxies, if any
	GetProxyResources() corev1.ResourceRequirements

	// GetInboundExternalAuthConfig returns the External Authentication configuration for incoming traffic, if any
	GetInboundExternalAuthConfig() auth.ExtAuthConfig

	// GetFeatureFlags returns OSM's feature flags
	GetFeatureFlags() configv1alpha1.FeatureFlags
}
