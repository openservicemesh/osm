package namespace

import (
	"k8s.io/client-go/tools/cache"

	"github.com/openservicemesh/osm/pkg/logger"
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
	IsMonitoredNamespace(string) bool
}
