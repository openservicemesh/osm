package configurator

import (
	"k8s.io/client-go/tools/cache"

	"github.com/open-service-mesh/osm/pkg/logger"
)

var (
	log = logger.New("configurator")
)

// Config is a struct with common global config settings.
type Config struct {
	// OSMNamespace is the Kubernetes namespace in which the OSM controller is installed.
	OSMNamespace string

	// EnablePrometheus enables Prometheus metrics integration when true
	EnablePrometheus bool

	// EnableTracing enables Zipkin tracing when true.
	EnableTracing bool
}

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
	// GetOSMNamespace returns the namespace in which OSM controller pod resides.
	GetOSMNamespace() string

	// GetConfigMap returns the ConfigMap in pretty JSON (human readable).
	GetConfigMap() ([]byte, error)

	// IsPermissiveTrafficPolicyMode determines whether we are in "allow-all" mode or SMI policy (block by default) mode.
	IsPermissiveTrafficPolicyMode() bool

	// IsEgressEnabled determines whether egress is globally enabled in the mesh or not.
	IsEgressEnabled() bool

	// GetAnnouncementsChannel returns a channel, which is used to announce when changes have been made to the OSM ConfigMap.
	GetAnnouncementsChannel() <-chan interface{}
}
