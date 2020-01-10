package catalog

import (
	"sync"

	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"

	"github.com/deislabs/smc/pkg/mesh"
)

// ServiceCatalog is the struct for the service catalog
type ServiceCatalog struct {
	sync.Mutex
	servicesCache      map[mesh.ServiceName][]mesh.IP
	endpointsProviders []mesh.EndpointsProvider
	meshTopology       mesh.Topology
}

// ServiceCataloger is the mechanism by which the Service Mesh controller discovers all Envoy proxies connected to the catalog.
type ServiceCataloger interface {

	// GetWeightedServices is deprecated
	// Deprecated: this needs to be removed
	GetWeightedServices() (map[mesh.ServiceName][]mesh.WeightedService, error)

	// ListEndpoints constructs a DescoveryResponse with all endpoints the given Envoy proxy should be aware of.
	// The bool return value indicates whether there have been any changes since the last invocation of this function.
	ListEndpoints(mesh.ClientIdentity) (v2.DiscoveryResponse, bool, error)

	// RegisterNewEndpoint adds a newly connected Envoy proxy to the list of self-announced endpoints for a service.
	RegisterNewEndpoint(mesh.ClientIdentity)

	// ListEndpointsProviders retrieves the full list of endpoints providers registered with Service Catalog so far.
	ListEndpointsProviders() []mesh.EndpointsProvider

	// RegisterEndpointsProvider adds a new endpoints provider to the list within the Service Catalog.
	RegisterEndpointsProvider(mesh.EndpointsProvider) error

	// GetAnnouncementChannel returns an instance of a channel, which notifies the system of an event requiring the execution of ListEndpoints.
	GetAnnouncementChannel() chan struct{}
}
