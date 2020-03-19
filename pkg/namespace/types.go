package namespace

import (
	"k8s.io/client-go/tools/cache"
)

type friendlyName string

// InformerCollection is a struct of the Kubernetes informers used in OSM
type InformerCollection struct {
	MonitorNamespaces cache.SharedIndexInformer
}

// CacheCollection is a struct of the Kubernetes caches used in OSM
type CacheCollection struct {
	MonitorNamespaces cache.Store
}

// Client is a struct for all components necessary to connect to and maintain state of a Kubernetes cluster.
type Client struct {
	informers   *InformerCollection
	caches      *CacheCollection
	cacheSynced chan interface{}
}

// Controller is the controller interface for K8s namespaces
type Controller interface {
	IsMonitoredNamespace(string) bool
}
