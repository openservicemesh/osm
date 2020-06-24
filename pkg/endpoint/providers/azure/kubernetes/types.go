package azure

import (
	"github.com/open-service-mesh/osm/pkg/namespace"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"github.com/open-service-mesh/osm/pkg/configurator"
	"github.com/open-service-mesh/osm/pkg/logger"
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
	caches              *CacheCollection
	cacheSynced         chan interface{}
	kubeClient          kubernetes.Interface
	informers           *InformerCollection
	providerIdent       string
	announcements       chan interface{}
	namespaceController namespace.Controller
	configurator        configurator.Configurator
}
