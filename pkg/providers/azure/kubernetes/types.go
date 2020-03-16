package azure

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

// InformerCollection is a struct of the Kubernetes informers used in OSM
type InformerCollection struct {
	AzureResource cache.SharedIndexInformer
}

// CacheCollection is a struct of the Kubernetes caches used in OSM
type CacheCollection struct {
	AzureResource cache.Store
}

// Client is a struct for all components necessary to connect to and maintain state of a Kubernetes cluster.
type Client struct {
	caches        *CacheCollection
	cacheSynced   chan interface{}
	kubeClient    kubernetes.Interface
	informers     *InformerCollection
	providerIdent string
	announcements chan interface{}
	namespaces    map[string]struct{}
}
