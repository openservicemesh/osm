package subm

import (
	"github.com/openservicemesh/osm/pkg/smi"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"github.com/openservicemesh/osm/pkg/logger"
	"github.com/openservicemesh/osm/pkg/namespace"
)

var (
	log = logger.New("subm-provider")
)

// InformerCollection is a struct of the Submariner informers used in OSM
type InformerCollection struct {
	ServiceImports   cache.SharedIndexInformer
}

// CacheCollection is a struct of the Submariner caches used in OSM
type CacheCollection struct {
	ServiceImports   cache.Store
}

// Client is a struct for all components necessary to connect to and maintain state of a Kubernetes cluster.
type Client struct {
	caches              *CacheCollection
	cacheSynced         chan interface{}
	providerIdent       string
	clusterId           string
	meshSpec            smi.MeshSpec
	kubeClient          kubernetes.Interface
	informers           *InformerCollection
	announcements       chan interface{}
	namespaceController namespace.Controller
}
