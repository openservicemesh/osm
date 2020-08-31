package smi

import (
	target "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/access/v1alpha2"
	spec "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/specs/v1alpha3"
	split "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/split/v1alpha2"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"

	backpressure "github.com/openservicemesh/osm/experimental/pkg/apis/policy/v1alpha1"
	"github.com/openservicemesh/osm/pkg/logger"
	"github.com/openservicemesh/osm/pkg/namespace"
	"github.com/openservicemesh/osm/pkg/service"
)

var (
	log = logger.New("mesh-spec")
)

// InformerCollection is a struct of the Kubernetes informers used in OSM
type InformerCollection struct {
	Services       cache.SharedIndexInformer
	TrafficSplit   cache.SharedIndexInformer
	HTTPRouteGroup cache.SharedIndexInformer
	TCPRoute       cache.SharedIndexInformer
	TrafficTarget  cache.SharedIndexInformer
	Backpressure   cache.SharedIndexInformer
}

// CacheCollection is a struct of the Kubernetes caches used in OSM
type CacheCollection struct {
	Services       cache.Store
	TrafficSplit   cache.Store
	HTTPRouteGroup cache.Store
	TCPRoute       cache.Store
	TrafficTarget  cache.Store
	Backpressure   cache.Store
}

// Client is a struct for all components necessary to connect to and maintain state of a Kubernetes cluster.
type Client struct {
	caches              *CacheCollection
	cacheSynced         chan interface{}
	providerIdent       string
	informers           *InformerCollection
	announcements       chan interface{}
	osmNamespace        string
	namespaceController namespace.Controller
}

// MeshSpec is an interface declaring functions, which provide the specs for a service mesh declared with SMI.
type MeshSpec interface {
	// ListTrafficSplits lists SMI TrafficSplit resources
	ListTrafficSplits() []*split.TrafficSplit

	// ListTrafficSplitServices lists WeightedServices for the services specified in TrafficSplit SMI resources
	ListTrafficSplitServices() []service.WeightedService

	// ListServiceAccounts lists ServiceAccount resources specified in SMI TrafficTarget resources
	ListServiceAccounts() []service.K8sServiceAccount

	// GetService fetches a Kubernetes Service resource for the given MeshService
	GetService(service.MeshService) *corev1.Service

	// ListServices Lists Kubernets Service resources that are part of monitored namespaces
	ListServices() []*corev1.Service

	// ListHTTPTrafficSpecs lists SMI HTTPRouteGroup resources
	ListHTTPTrafficSpecs() []*spec.HTTPRouteGroup

	// ListTCPTrafficSpecs lists SMI TCPRoute resources
	ListTCPTrafficSpecs() []*spec.TCPRoute

	// ListTrafficTargets lists SMI TrafficTarget resources
	ListTrafficTargets() []*target.TrafficTarget

	// GetBackpressurePolicy fetches the Backpressure policy for the MeshService
	GetBackpressurePolicy(service.MeshService) *backpressure.Backpressure

	// GetAnnouncementsChannel returns the channel on which SMI client makes announcements
	GetAnnouncementsChannel() <-chan interface{}
}
