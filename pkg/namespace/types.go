package namespace

import (
	"k8s.io/client-go/tools/cache"
)

// Client is a struct for all components necessary to connect to and maintain state of a Kubernetes cluster.
type Client struct {
	informer    cache.SharedIndexInformer
	cache       cache.Store
	cacheSynced chan interface{}
}

// Controller is the controller interface for K8s namespaces
type Controller interface {
	IsMonitoredNamespace(string) bool
}
