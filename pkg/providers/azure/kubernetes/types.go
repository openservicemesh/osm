package azure

import (
	"github.com/eapache/channels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

// InformerCollection is a struct of the Kubernetes informers used in SMC
type InformerCollection struct {
	AzureResource cache.SharedIndexInformer
}

// CacheCollection is a struct of the Kubernetes caches used in SMC
type CacheCollection struct {
	AzureResource cache.Store
}

// Client is a struct for all components necessary to connect to and maintain state of a Kubernetes cluster.
type Client struct {
	caches        *CacheCollection
	cacheSynced   chan interface{}
	kubeClient    kubernetes.Interface
	informers     *InformerCollection
	announcements *channels.RingChannel
	providerIdent string
}
