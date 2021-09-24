// Package smi implements the Service Mesh Interface (SMI) kubernetes client to observe and retrieve information
// regarding SMI traffic resources.
package smi

import (
	"k8s.io/client-go/tools/cache"

	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/k8s"
	"github.com/openservicemesh/osm/pkg/logger"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/smi/specs"
)

var (
	log = logger.New("smi-mesh-spec")
)

// informerCollection is a struct of the Kubernetes informers used for SMI resources
type informerCollection struct {
	TrafficSplit   cache.SharedIndexInformer
	HTTPRouteGroup cache.SharedIndexInformer
	TCPRoute       cache.SharedIndexInformer
	TrafficTarget  cache.SharedIndexInformer
}

// cacheCollection is a struct of the Kubernetes caches used for SMI resources
type cacheCollection struct {
	TrafficSplit   cache.Store
	HTTPRouteGroup cache.Store
	TCPRoute       cache.Store
	TrafficTarget  cache.Store
}

// client is a type that implements the smiSpecs.MeshSpec interface related to Kubernetes SMI resources
type client struct {
	caches         *cacheCollection
	providerIdent  string
	informers      *informerCollection
	osmNamespace   string
	kubeController k8s.Controller
}

// WithTrafficTargetDestination applies a filter based on the destination service account to the TrafficTarget lister
func WithTrafficTargetDestination(d identity.K8sServiceAccount) specs.TrafficTargetListOption {
	return func(o *specs.TrafficTargetListOpt) {
		o.Destination = d
	}
}

// WithTrafficSplitApexService applies a filter based on the apex service to the TrafficSplit lister
func WithTrafficSplitApexService(s service.MeshService) specs.TrafficSplitListOption {
	return func(o *specs.TrafficSplitListOpt) {
		o.ApexService = s
	}
}

// WithTrafficSplitBackendService applies a filter based on the backend service to the TrafficSplit lister
func WithTrafficSplitBackendService(s service.MeshService) specs.TrafficSplitListOption {
	return func(o *specs.TrafficSplitListOpt) {
		o.BackendService = s
	}
}
