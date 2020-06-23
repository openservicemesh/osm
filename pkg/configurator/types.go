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


// Client is the k8s client struct for the OSMConfig CRD.
type Client struct {
	configCRDName      string
	configCRDNamespace string
	informer           cache.SharedIndexInformer
	cache              cache.Store
	cacheSynced        chan interface{}
	announcements      chan interface{}
}

// Configurator is the controller interface for K8s namespaces
type Configurator interface {
	IsMonitoredNamespace(string) bool
}
