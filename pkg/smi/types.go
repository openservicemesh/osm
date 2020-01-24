package smi

import (
	"github.com/eapache/channels"
	"k8s.io/client-go/tools/cache"
)

type friendlyName string

// InformerCollection is a struct of the Kubernetes informers used in SMC
// todo (snchh) : might need a separate informer for the TCP TrafficSpec
type InformerCollection struct {
	Services      cache.SharedIndexInformer
	TrafficSplit  cache.SharedIndexInformer
	TrafficSpec   cache.SharedIndexInformer
	TrafficTarget cache.SharedIndexInformer
}

// CacheCollection is a struct of the Kubernetes caches used in SMC
type CacheCollection struct {
	Services      cache.Store
	TrafficSplit  cache.Store
	TrafficSpec   cache.Store
	TrafficTarget cache.Store
}

// Client is a struct for all components necessary to connect to and maintain state of a Kubernetes cluster.
type Client struct {
	caches        *CacheCollection
	cacheSynced   chan interface{}
	providerIdent string
	informers     *InformerCollection
	announceChan  *channels.RingChannel
}
