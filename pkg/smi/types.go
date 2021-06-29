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
	ListTrafficSplits() []*split.TrafficSplit

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

	// ListTrafficTargets lists SMI TrafficTarget resources
	ListTrafficTargets() []*access.TrafficTarget
}
