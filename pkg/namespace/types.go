package namespace

import (
	"github.com/openservicemesh/osm/pkg/logger"
	"k8s.io/client-go/tools/cache"
)

var (
	log = logger.New("kube-namespace")
)

// Client is a struct for all components necessary to connect to and maintain state of a Kubernetes cluster.
type Client struct {
	informer      cache.SharedIndexInformer
	cache         cache.Store
	cacheSynced   chan interface{}
	announcements chan interface{}
}

// Controller is the controller interface for K8s namespaces
type Controller interface {
	// IsMonitoredNamespace returns whether a namespace with the given name is being monitored
	// by the mesh
	IsMonitoredNamespace(string) bool

	// ListMonitoredNamespaces returns the namespaces monitored by the mesh
	ListMonitoredNamespaces() ([]string, error)

	// GetAnnouncementsChannel returns the channel on which namespace makes announcements
	GetAnnouncementsChannel() <-chan interface{}
}
