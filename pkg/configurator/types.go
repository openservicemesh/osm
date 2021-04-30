// Package configurator implements the Configurator interface that provides APIs to retrieve OSM control plane configurations.
package configurator

import (
	"time"

	"k8s.io/client-go/tools/cache"

	"github.com/openservicemesh/osm/pkg/logger"
)

var (
	log = logger.New("configurator")
)

// CRDClient is the k8s client struct for the MeshConfig CRD. The feature is in experimental stage.
type CRDClient struct {
	// TODO: rename it to `client`
	osmNamespace   string
	informer       cache.SharedIndexInformer
	cache          cache.Store
	cacheSynced    chan interface{}
	meshConfigName string
}

// Configurator is the controller interface for K8s namespaces
type Configurator interface {
	// GetOSMNamespace returns the namespace in which OSM controller pod resides
	GetOSMNamespace() string

	// GetMeshConfigJSON returns the MeshConfig in pretty JSON (human readable)
	GetMeshConfigJSON() ([]byte, error)

	// IsPermissiveTrafficPolicyMode determines whether we are in "allow-all" mode or SMI policy (block by default) mode
	IsPermissiveTrafficPolicyMode() bool

	// IsEgressEnabled determines whether egress is globally enabled in the mesh or not
	IsEgressEnabled() bool

	// IsDebugServerEnabled determines whether osm debug HTTP server is enabled
	IsDebugServerEnabled() bool

	// IsPrometheusScrapingEnabled determines whether Prometheus is enabled for scraping metrics
	IsPrometheusScrapingEnabled() bool

	// IsTracingEnabled returns whether tracing is enabled
	IsTracingEnabled() bool

	// GetTracingHost is the host to which we send tracing spans
	GetTracingHost() string

	// GetTracingPort returns the tracing listener port
	GetTracingPort() uint32

	// GetTracingEndpoint returns the collector endpoint
	GetTracingEndpoint() string

	// UseHTTPSIngress determines whether protocol used for traffic from ingress to backend pods should be HTTPS.
	UseHTTPSIngress() bool

	// GetMaxDataPlaneConnections returns the max data plane connections allowed, 0 if disabled
	GetMaxDataPlaneConnections() int

	// GetEnvoyLogLevel returns the envoy log level
	GetEnvoyLogLevel() string

	// GetEnvoyImage returns the envoy image
	GetEnvoyImage() string

	// GetServiceCertValidityPeriod returns the validity duration for service certificates
	GetServiceCertValidityPeriod() time.Duration

	// GetOutboundIPRangeExclusionList returns the list of IP ranges of the form x.x.x.x/y to exclude from outbound sidecar interception
	GetOutboundIPRangeExclusionList() []string

	// GetOutboundPortExclusionList returns the list of ports to exclude from outbound sidecar interception
	GetOutboundPortExclusionList() []string

	// IsPrivilegedInitContainer determines whether init containers should be privileged
	IsPrivilegedInitContainer() bool

	// GetConfigResyncInterval returns the duration for resync interval.
	// If error or non-parsable value, returns 0 duration
	GetConfigResyncInterval() time.Duration
}
