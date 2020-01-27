package catalog

import (
	"sync"

	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"

	"github.com/deislabs/smc/pkg/certificate"
	"github.com/deislabs/smc/pkg/endpoint"
	"github.com/deislabs/smc/pkg/envoy"
	"github.com/deislabs/smc/pkg/smi"
)

// MeshCataloger is the mechanism by which the Service Mesh controller discovers all Envoy proxies connected to the catalog.
type MeshCataloger interface {
	// ListEndpoints constructs a map of service to weighted handlers with all endpoints the given Envoy proxy should be aware of.
	ListEndpoints(smi.ClientIdentity) (map[endpoint.ServiceName][]endpoint.WeightedService, error)

	// ListTrafficRoutes constructs a DiscoveryResponse with all traffic routes the given Envoy proxy should be aware of.
	ListTrafficRoutes(smi.ClientIdentity) (resp *v2.DiscoveryResponse, err error)

	// RegisterProxy registers a newly connected proxy with the service mesh catalog.
	RegisterProxy(envoy.Proxyer)

	// GetAnnouncementChannel returns an instance of a channel, which notifies the system of an event requiring the execution of ListEndpoints.
	GetAnnouncementChannel() chan interface{}

	// GetCertificateForService returns the SSL Certificate for the given service.
	// This certificate will be used for service-to-service mTLS.
	GetCertificateForService(endpoint.ServiceName) (certificate.Certificater, error)
}

// MeshCatalog is the struct for the service catalog
type MeshCatalog struct {
	sync.Mutex

	announcements chan interface{}

	endpointsProviders []endpoint.Provider
	meshSpec           smi.MeshSpec
	certManager        certificate.Manager

	// Caches
	servicesCache    map[endpoint.ServiceName][]endpoint.Endpoint
	certificateCache map[endpoint.ServiceName]certificate.Certificater
	connectedProxies []envoy.Proxyer
}
