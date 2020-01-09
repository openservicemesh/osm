package mesh

import (
	"github.com/deislabs/smi-sdk-go/pkg/apis/split/v1alpha2"
	envoy "github.com/envoyproxy/go-control-plane/envoy/api/v2"
)

type ClientIdentity string

// ServiceCataloger is the mechanism by which the Service Mesh controller discovers all Envoy proxies connected to the catalog.
type ServiceCataloger interface {

	// GetWeightedServices is deprecated
	// Deprecated: this needs to be removed
	GetWeightedServices() (map[ServiceName][]WeightedService, error)

	// ListEndpoints constructs a DescoveryResponse with all endpoints the given Envoy proxy should be aware of.
	// The bool return value indicates whether there have been any changes since the last invocation of this function.
	ListEndpoints(ClientIdentity) (envoy.DiscoveryResponse, bool, error)

	// RegisterNewEndpoint adds a newly connected Envoy proxy to the list of self-announced endpoints for a service.
	RegisterNewEndpoint(ClientIdentity)

	// ListEndpointsProviders retrieves the full list of endpoints providers registered with Service Catalog so far.
	ListEndpointsProviders() []EndpointsProvider

	// RegisterEndpointsProvider adds a new endpoints provider to the list within the Service Catalog.
	RegisterEndpointsProvider(EndpointsProvider) error

	// GetAnnouncementChannel returns an instance of a channel, which notifies the system of an event requiring the execution of ListEndpoints.
	GetAnnouncementChannel() chan struct{}
}

// ServiceName is a type for a service name
type ServiceName string

func (sn ServiceName) String() string {
	return string(sn)
}

// WeightedService is a struct of a delegated service backing a target service
type WeightedService struct {
	ServiceName ServiceName `json:"service_name:omitempty"`
	Weight      int         `json:"weight:omitempty"`
	IPs         []IP        `json:"ips:omitempty"`
}

// IP is an IP address
type IP string

// EndpointProvider is an interface to be implemented by components abstracting Kubernetes, Azure, and other compute/cluster providers
type EndpointsProvider interface {
	// Retrieve the IP addresses comprising the ServiceName.
	GetIPs(ServiceName) []IP
	GetID() string
	Run(<-chan struct{}) error
}

// Topology is an interface declaring functions, which provide the topology of a service mesh declared with SMI.
type Topology interface {
	// ListTrafficSplits lists TrafficSplit SMI resources.
	ListTrafficSplits() []*v1alpha2.TrafficSplit

	// ListServices fetches all services declared with SMI Spec.
	ListServices() []ServiceName

	// GetComputeIDForService is deprecated
	// Deprecated: this needs to be removed
	GetComputeIDForService(ServiceName) []ComputeID
}

// AzureID is a string type alias, which is the URI of a unique Azure cloud resource.
type AzureID string

// KubernetesID is a struct type, which points to a uniquely identifiable Kubernetes cluster.
type KubernetesID struct {
	ClusterID string
	Namespace string
	Service   string
}

// ComputeID is a struct, which contains the unique IDs of hte compute clusters where certain service may have Endpoints in.
type ComputeID struct {
	AzureID
	KubernetesID
}
