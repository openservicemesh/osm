package configurator

import (
	"k8s.io/client-go/tools/cache"
	"time"

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

	// IsPrometheusScrapingEnabled determines whether Prometheus is enabled for scraping metrics
	IsPrometheusScrapingEnabled() bool

	// IsZipkinTracingEnabled determines whether Zipkin tracing is enabled
	IsZipkinTracingEnabled() bool

	// GetZipkinHost is the host to which we send Zipkin spans
	GetZipkinHost() string

	// GetZipkinPort returns the Zipkin port
	GetZipkinPort() uint32

	// GetZipkinEndpoint returns the Zipkin endpoint
	GetZipkinEndpoint() string

	// GetMeshCIDRRanges returns a list of mesh CIDR ranges
	GetMeshCIDRRanges() []string

	// UseHTTPSIngress determines whether protocol used for traffic from ingress to backend pods should be HTTPS.
	UseHTTPSIngress() bool

	// GetGetControlPlaneCertValidityPeriod returns the duration of validity for the Envoy to XDS certificate
	GetGetControlPlaneCertValidityPeriod() time.Duration

	// GetAnnouncementsChannel returns a channel, which is used to announce when changes have been made to the OSM ConfigMap
	GetAnnouncementsChannel() <-chan interface{}
}
