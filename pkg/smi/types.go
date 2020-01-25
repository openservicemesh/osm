package smi

import (
	TrafficTarget "github.com/deislabs/smi-sdk-go/pkg/apis/access/v1alpha1"
	TrafficSpec "github.com/deislabs/smi-sdk-go/pkg/apis/specs/v1alpha1"
	TrafficSplit "github.com/deislabs/smi-sdk-go/pkg/apis/split/v1alpha2"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"

	"github.com/deislabs/smc/pkg/endpoint"
)

type friendlyName string

// InformerCollection is a struct of the Kubernetes informers used in SMC
type InformerCollection struct {
	Services     cache.SharedIndexInformer
	TrafficSplit cache.SharedIndexInformer
	TrafficSpec   cache.SharedIndexInformer
	TrafficTarget cache.SharedIndexInformer
}

// CacheCollection is a struct of the Kubernetes caches used in SMC
type CacheCollection struct {
	Services     cache.Store
	TrafficSplit cache.Store
	TrafficSpec   cache.Store
	TrafficTarget cache.Store
}

// Client is a struct for all components necessary to connect to and maintain state of a Kubernetes cluster.
type Client struct {
	caches        *CacheCollection
	cacheSynced   chan interface{}
	providerIdent string
	informers     *InformerCollection
	announcements chan interface{}
}

// ClientIdentity is the identity of an Envoy proxy connected to the Service Mesh Controller.
type ClientIdentity string

// MeshSpec is an interface declaring functions, which provide the spec of a service mesh declared with SMI.
type MeshSpec interface {
	// ListTrafficSplits lists TrafficSplit SMI resources.
	ListTrafficSplits() []*TrafficSplit.TrafficSplit

	// ListServices fetches all services declared with SMI Spec.
	ListServices() []endpoint.ServiceName

	// GetService fetches a specific service declared in SMI.
	GetService(endpoint.ServiceName) (service *v1.Service, exists bool, err error)

	// ListHTTPTrafficSpecs lists TrafficSpec SMI resources.
	ListHTTPTrafficSpecs() []*TrafficSpec.HTTPRouteGroup

	// ListTrafficTargets lists TrafficTarget SMI resources.
	ListTrafficTargets() []*TrafficTarget.TrafficTarget
}