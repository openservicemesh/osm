package kube

import (
	"github.com/eapache/channels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/utils"
)

// InformerCollection is a struct of the Kubernetes informers used in SMC
type InformerCollection struct {
	Endpoints    cache.SharedIndexInformer
	Services     cache.SharedIndexInformer
	Pods         cache.SharedIndexInformer
	TrafficSplit cache.SharedIndexInformer
}

// CacheCollection is a struct of the Kubernetes caches used in SMC
type CacheCollection struct {
	Endpoints    cache.Store
	Services     cache.Store
	Pods         cache.Store
	TrafficSplit cache.Store
}

// KubernetesProvider is a struct of the Kubernetes config and components used in SMC
type KubernetesProvider struct {
	kubeClient        kubernetes.Interface
	informers         *InformerCollection
	Caches            *CacheCollection
	ingressSecretsMap utils.ThreadsafeMultiMap
	announceChan      *channels.RingChannel
	CacheSynced       chan interface{}
}
