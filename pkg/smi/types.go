// Package smi implements the Service Mesh Interface (SMI) kubernetes client to observe and retrieve information
// regarding SMI traffic resources.
package smi

import (
	access "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/access/v1alpha3"
	spec "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/specs/v1alpha4"
	split "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/split/v1alpha2"

	"k8s.io/client-go/tools/cache"

	"github.com/openservicemesh/osm/pkg/announcements"
	k8s "github.com/openservicemesh/osm/pkg/kubernetes"
	"github.com/openservicemesh/osm/pkg/logger"
	"github.com/openservicemesh/osm/pkg/service"
)

var (
	log = logger.New("mesh-spec")
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

// Client is a struct for all components necessary to connect to and maintain state of a Kubernetes cluster.
type Client struct {
	caches         *cacheCollection
	cacheSynced    chan interface{}
	providerIdent  string
	informers      *informerCollection
	announcements  chan announcements.Announcement
	osmNamespace   string
	kubeController k8s.Controller
}

// MeshSpec is an interface declaring functions, which provide the specs for a service mesh declared with SMI.
type MeshSpec interface {
	// ListTrafficSplits lists SMI TrafficSplit resources
	ListTrafficSplits() []*split.TrafficSplit

	// ListServiceAccounts lists ServiceAccount resources specified in SMI TrafficTarget resources
	ListServiceAccounts() []service.K8sServiceAccount

	// ListHTTPTrafficSpecs lists SMI HTTPRouteGroup resources
	ListHTTPTrafficSpecs() []*spec.HTTPRouteGroup

	// ListTCPTrafficSpecs lists SMI TCPRoute resources
	ListTCPTrafficSpecs() []*spec.TCPRoute

	// GetTCPRoute returns an SMI TCPRoute resource given its name of the form <namespace>/<name>
	GetTCPRoute(string) *spec.TCPRoute

	// ListTrafficTargets lists SMI TrafficTarget resources
	ListTrafficTargets() []*access.TrafficTarget
}
