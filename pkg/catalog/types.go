package catalog

import (
	"sync"

	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"k8s.io/client-go/util/certificate"

	"github.com/deislabs/smc/pkg/endpoint"
	"github.com/deislabs/smc/pkg/envoy"
	"github.com/deislabs/smc/pkg/smi"
)

// MeshCatalog is the struct for the service catalog
type MeshCatalog struct {
	sync.Mutex

	announcements chan struct{}

	endpointsProviders []endpoint.Provider
	meshSpec           smi.MeshSpec
	certManager        certificate.Manager

	// Caches
	servicesCache map[endpoint.ServiceName][]endpoint.Endpoint
	// certificateCache map[endpoint.ServiceName]certificate.Certificater
	connectedProxies []envoy.Proxyer
}

// MeshCataloger is the mechanism by which the Service Mesh controller discovers all Envoy proxies connected to the catalog.
type MeshCataloger interface {
	// ListEndpoints constructs a map of service to weighted handlers with all endpoints the given Envoy proxy should be aware of.
	// The bool return value indicates whether there have been any changes since the last invocation of this function.
	ListEndpoints(smi.ClientIdentity) (map[endpoint.ServiceName][]endpoint.WeightedService, error)

	// ListTrafficRoutes constructs a DiscoveryResponse with all traffic routes the given Envoy proxy should be aware of.
	// The bool return value indicates whether there have been any changes since the last invocation of this function.
	ListTrafficRoutes(smi.ClientIdentity) (resp *v2.DiscoveryResponse, hasChanged bool, err error)

	// RegisterNewEndpoint adds a newly connected Envoy proxy to the list of self-announced endpoints for a service.
	RegisterNewEndpoint(smi.ClientIdentity)

	// ListEndpointsProviders retrieves the full list of endpoints providers registered with Service Catalog so far.
	ListEndpointsProviders() []endpoint.Provider

	// GetAnnouncementChannel returns an instance of a channel, which notifies the system of an event requiring the execution of ListEndpoints.
	GetAnnouncementChannel() chan struct{}

	// RegisterProxy registers a newly connected proxy with the service mesh catalog.
	RegisterProxy(envoy.Proxyer)
}
