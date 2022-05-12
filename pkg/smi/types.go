// Package smi implements the Service Mesh Interface (SMI) kubernetes client to observe and retrieve information
// regarding SMI traffic resources.
package smi

import (
	access "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/access/v1alpha3"
	spec "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/specs/v1alpha4"
	split "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/split/v1alpha2"

	"k8s.io/client-go/tools/cache"

	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/k8s"
	"github.com/openservicemesh/osm/pkg/logger"
	"github.com/openservicemesh/osm/pkg/service"
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

// client is a type that implements the smi.MeshSpec interface related to Kubernetes SMI resources
type client struct {
	caches         *cacheCollection
	providerIdent  string
	informers      *informerCollection
	osmNamespace   string
	kubeController k8s.Controller
}

// MeshSpec is an interface declaring functions, which provide the specs for a service mesh declared with SMI.
type MeshSpec interface {
	// ListTrafficSplits lists SMI TrafficSplit resources
	ListTrafficSplits(...TrafficSplitListOption) []*split.TrafficSplit

	// ListServiceAccounts lists ServiceAccount resources specified in SMI TrafficTarget resources
	ListServiceAccounts() []identity.K8sServiceAccount

	// ListHTTPTrafficSpecs lists SMI HTTPRouteGroup resources
	ListHTTPTrafficSpecs() []*spec.HTTPRouteGroup

	// GetHTTPRouteGroup returns an SMI HTTPRouteGroup resource given its name of the form <namespace>/<name>
	GetHTTPRouteGroup(string) *spec.HTTPRouteGroup

	// ListTCPTrafficSpecs lists SMI TCPRoute resources
	ListTCPTrafficSpecs() []*spec.TCPRoute

	// GetTCPRoute returns an SMI TCPRoute resource given its name of the form <namespace>/<name>
	GetTCPRoute(string) *spec.TCPRoute

	// ListTrafficTargets lists SMI TrafficTarget resources. An optional filter can be applied to filter the
	// returned list
	ListTrafficTargets(...TrafficTargetListOption) []*access.TrafficTarget
}

// TrafficTargetListOpt specifies the options used to filter TrafficTarget objects as a part of its lister
type TrafficTargetListOpt struct {
	Destination identity.K8sServiceAccount
}

// TrafficTargetListOption is a function type that implements filters on TrafficTarget lister
type TrafficTargetListOption func(o *TrafficTargetListOpt)

// WithTrafficTargetDestination applies a filter based on the destination service account to the TrafficTarget lister
func WithTrafficTargetDestination(d identity.K8sServiceAccount) TrafficTargetListOption {
	return func(o *TrafficTargetListOpt) {
		o.Destination = d
	}
}

// TrafficSplitListOpt specifies the options used to filter TrafficSplit objects as a part of its lister
type TrafficSplitListOpt struct {
	ApexService    service.MeshService
	BackendService service.MeshService
}

// TrafficSplitListOption is a function type that implements filters on the TrafficSplit lister
type TrafficSplitListOption func(o *TrafficSplitListOpt)

// WithTrafficSplitApexService applies a filter based on the apex service to the TrafficSplit lister
func WithTrafficSplitApexService(s service.MeshService) TrafficSplitListOption {
	return func(o *TrafficSplitListOpt) {
		o.ApexService = s
	}
}

// WithTrafficSplitBackendService applies a filter based on the backend service to the TrafficSplit lister
func WithTrafficSplitBackendService(s service.MeshService) TrafficSplitListOption {
	return func(o *TrafficSplitListOpt) {
		o.BackendService = s
	}
}
