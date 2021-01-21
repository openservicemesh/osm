package smi

import (
	access "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/access/v1alpha3"
	spec "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/specs/v1alpha4"
	split "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/split/v1alpha2"

	"k8s.io/client-go/tools/cache"

	backpressure "github.com/openservicemesh/osm/experimental/pkg/apis/policy/v1alpha1"
	"github.com/openservicemesh/osm/pkg/announcements"
	k8s "github.com/openservicemesh/osm/pkg/kubernetes"
	"github.com/openservicemesh/osm/pkg/logger"
	"github.com/openservicemesh/osm/pkg/service"
)

var (
	log = logger.New("mesh-spec")
)

// InformerCollection is a struct of the Kubernetes informers used in OSM
type InformerCollection struct {
	TrafficSplit   cache.SharedIndexInformer
	HTTPRouteGroup cache.SharedIndexInformer
	TCPRoute       cache.SharedIndexInformer
	TrafficTarget  cache.SharedIndexInformer
	Backpressure   cache.SharedIndexInformer
}

// CacheCollection is a struct of the Kubernetes caches used in OSM
type CacheCollection struct {
	TrafficSplit   cache.Store
	HTTPRouteGroup cache.Store
	TCPRoute       cache.Store
	TrafficTarget  cache.Store
	Backpressure   cache.Store
}

// Client is a struct for all components necessary to connect to and maintain state of a Kubernetes cluster.
type Client struct {
	caches         *CacheCollection
	cacheSynced    chan interface{}
	providerIdent  string
	informers      *InformerCollection
	announcements  chan announcements.Announcement
	osmNamespace   string
	kubeController k8s.Controller
}

// MeshSpec is an interface declaring functions, which provide the specs for a service mesh declared with SMI.
type MeshSpec interface {
	// ListTrafficSplits lists SMI TrafficSplit resources
	ListTrafficSplits() []*split.TrafficSplit

	// ListTrafficSplitServices lists WeightedServices for the services specified in TrafficSplit SMI resources
	ListTrafficSplitServices() []service.WeightedService

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

	// GetBackpressurePolicy fetches the Backpressure policy for the MeshService
	GetBackpressurePolicy(service.MeshService) *backpressure.Backpressure
}
