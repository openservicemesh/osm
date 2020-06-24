package configurator

import (
	"sync"
	"time"

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

// ConfigMapWatcher is the k8s client struct for the OSM Config.
type ConfigMapWatcher struct {
	sync.Mutex

	refreshConfigMap time.Duration
	osmNamespace     string
	announcements    chan interface{}

	configMap *configMap
}

// Configurator is the controller interface for K8s namespaces
type Configurator interface {
	// GetOSMNamespace returns the namespace in which OSM controller pod resides.
	GetOSMNamespace() string

	// GetConfigMap returns the ConfigMap
	GetConfigMap() ([]byte, error)
}
