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
	announcements    chan interface{}
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

	// GetAnnouncementsChannel returns a channel, which is used to announce when changes have been made to the OSM ConfigMap
	GetAnnouncementsChannel() <-chan interface{}

	// GetServiceCertValidityPeriod returns the validity duration for service certificates
	GetServiceCertValidityPeriod() time.Duration
}
