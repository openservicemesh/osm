package kube

import (
	"reflect"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"github.com/open-service-mesh/osm/pkg/namespace"
	"github.com/open-service-mesh/osm/pkg/utils"
)

type friendlyName string

type empty struct{}

var packageName = utils.GetLastChunkOfSlashed(reflect.TypeOf(empty{}).PkgPath())

// InformerCollection is a struct of the Kubernetes informers used in OSM
type InformerCollection struct {
	Endpoints   cache.SharedIndexInformer
	Deployments cache.SharedIndexInformer
}

// CacheCollection is a struct of the Kubernetes caches used in OSM
type CacheCollection struct {
	Endpoints   cache.Store
	Deployments cache.Store
}

// Client is a struct for all components necessary to connect to and maintain state of a Kubernetes cluster.
type Client struct {
	caches              *CacheCollection
	cacheSynced         chan interface{}
	providerIdent       string
	kubeClient          kubernetes.Interface
	informers           *InformerCollection
	announcements       chan interface{}
	namespaceController namespace.Controller
}
