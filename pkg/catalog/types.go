package catalog

import (
	"sync"

	envoy "github.com/envoyproxy/go-control-plane/envoy/api/v2"

	"github.com/deislabs/smc/pkg/endpoint"
	smcEnvoy "github.com/deislabs/smc/pkg/envoy"
	"github.com/deislabs/smc/pkg/smi"
)

// MeshCatalog is the struct for the service catalog
type MeshCatalog struct {
	sync.Mutex

	servicesCache      map[endpoint.ServiceName][]endpoint.Endpoint
	endpointsProviders []endpoint.Provider

	meshSpec smi.MeshSpec
}

// MeshCataloger is the mechanism by which the Service Mesh controller discovers all Envoy proxies connected to the catalog.
type MeshCataloger interface {
	// ListEndpoints constructs a DiscoveryResponse with all endpoints the given Envoy proxy should be aware of.
	// The bool return value indicates whether there have been any changes since the last invocation of this function.
	ListEndpoints(smi.ClientIdentity) (resp *envoy.DiscoveryResponse, hasChanged bool, err error)

	// RegisterNewEndpoint adds a newly connected Envoy proxy to the list of self-announced endpoints for a service.
	RegisterNewEndpoint(smi.ClientIdentity)

	// ListEndpointsProviders retrieves the full list of endpoints providers registered with Service Catalog so far.
	ListEndpointsProviders() []endpoint.Provider

	// GetAnnouncementChannel returns an instance of a channel, which notifies the system of an event requiring the execution of ListEndpoints.
	GetAnnouncementChannel() chan struct{}

	// RegisterProxy registers a newly connected proxy with the service mesh catalog.
	RegisterProxy(smcEnvoy.Proxyer)
}
