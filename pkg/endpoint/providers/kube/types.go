package kube

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"github.com/openservicemesh/osm/pkg/announcements"
	k8s "github.com/openservicemesh/osm/pkg/kubernetes"
	"github.com/openservicemesh/osm/pkg/logger"
)

var (
	log = logger.New("kube-provider")
)

// InformerCollection is a struct of the Kubernetes informers used in OSM
type InformerCollection struct {
	Endpoints cache.SharedIndexInformer
	Pods      cache.SharedIndexInformer
}

// CacheCollection is a struct of the Kubernetes caches used in OSM
type CacheCollection struct {
	Endpoints cache.Store
	Pods      cache.Store
}

// Client is a struct for all components necessary to connect to and maintain state of a Kubernetes cluster.
type Client struct {
	caches         *CacheCollection
	cacheSynced    chan interface{}
	providerIdent  string
	kubeClient     kubernetes.Interface
	informers      *InformerCollection
	announcements  chan announcements.Announcement
	kubeController k8s.Controller
}
