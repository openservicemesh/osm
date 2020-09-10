package azure

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	k8s "github.com/openservicemesh/osm/pkg/kubernetes"
	"github.com/openservicemesh/osm/pkg/logger"
)

var (
	log = logger.New("azure-kube-provider")
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
	caches         *CacheCollection
	cacheSynced    chan interface{}
	kubeClient     kubernetes.Interface
	informers      *InformerCollection
	providerIdent  string
	announcements  chan interface{}
	kubeController k8s.Controller
}
