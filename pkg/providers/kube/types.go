package kube

import (
	"github.com/eapache/channels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

type friendlyName string

// InformerCollection is a struct of the Kubernetes informers used in SMC
type InformerCollection struct {
	Endpoints     cache.SharedIndexInformer
	Services      cache.SharedIndexInformer
	Pods          cache.SharedIndexInformer
	TrafficSplit  cache.SharedIndexInformer
	AzureResource cache.SharedIndexInformer
}

// CacheCollection is a struct of the Kubernetes caches used in SMC
type CacheCollection struct {
	Endpoints     cache.Store
	Services      cache.Store
	Pods          cache.Store
	TrafficSplit  cache.Store
	AzureResource cache.Store
}

// Client is a struct of the Kubernetes config and components used in SMC
type Client struct {
	kubeClient   kubernetes.Interface
	informers    *InformerCollection
	Caches       *CacheCollection
	announceChan *channels.RingChannel
	CacheSynced  chan interface{}
}
