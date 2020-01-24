package kube

import (
	"github.com/eapache/channels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

type friendlyName string

// InformerCollection is a struct of the Kubernetes informers used in SMC
type InformerCollection struct {
	Endpoints cache.SharedIndexInformer
}

// CacheCollection is a struct of the Kubernetes caches used in SMC
type CacheCollection struct {
	Endpoints cache.Store
}

// Client is a struct for all components necessary to connect to and maintain state of a Kubernetes cluster.
// Implements interfaces: ComputeProviderI
type Client struct {
	caches        *CacheCollection
	cacheSynced   chan interface{}
	providerIdent string
	kubeClient    kubernetes.Interface
	informers     *InformerCollection
	announcements  *channels.RingChannel
}
