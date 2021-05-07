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

// Client is the k8s client struct for the OSM Config.
type Client struct {
	osmNamespace     string
	osmConfigMapName string
	informer         cache.SharedIndexInformer
	cache            cache.Store
	cacheSynced      chan interface{}
}

// Configurator is the controller interface for K8s namespaces
type Configurator interface {
	// GetOSMNamespace returns the namespace in which OSM controller pod resides
	GetOSMNamespace() string

	// GetConfigMap returns the ConfigMap in pretty JSON (human readable)
	GetConfigMap() ([]byte, error)

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

	// GetEnvoyLogLevel returns the envoy log level
	GetEnvoyLogLevel() string

	// GetServiceCertValidityPeriod returns the validity duration for service certificates
	GetServiceCertValidityPeriod() time.Duration

	// GetOutboundIPRangeExclusionList returns the list of IP ranges of the form x.x.x.x/y to exclude from outbound sidecar interception
	GetOutboundIPRangeExclusionList() []string

	// IsPrivilegedInitContainer determines whether init containers should be privileged
	IsPrivilegedInitContainer() bool

	// GetConfigResyncInterval returns the duration for resync interval.
	// If error or non-parsable value, returns 0 duration
	GetConfigResyncInterval() time.Duration

	// GetInboundExternalAuthConfig returns the External Authentication configuration for incoming traffic, if any
	GetInboundExternalAuthConfig() ExternAuthConfig
}

// ExternAuthConfig implements a generic subset of External Authz to configure external authorization in envoy's format
type ExternAuthConfig struct {
	Enable           bool
	Address          string
	Port             uint16
	StatPrefix       string
	AuthzTimeout     time.Duration
	FailureModeAllow bool
}
