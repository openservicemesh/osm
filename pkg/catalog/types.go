package catalog

import (
	"sync"

	mapset "github.com/deckarep/golang-set"

	"github.com/deislabs/smc/pkg/certificate"
	"github.com/deislabs/smc/pkg/endpoint"
	"github.com/deislabs/smc/pkg/envoy"
	"github.com/deislabs/smc/pkg/smi"
)

// MeshCatalog is the struct for the service catalog
type MeshCatalog struct {
	endpointsProviders []endpoint.Provider
	meshSpec           smi.MeshSpec
	certManager        certificate.Manager

	servicesCache        map[endpoint.ServiceName][]endpoint.Endpoint
	servicesMutex        sync.Mutex
	certificateCache     map[endpoint.ServiceName]certificate.Certificater
	serviceAccountsCache map[endpoint.ServiceAccount][]endpoint.ServiceName
	virtualServicesCache map[endpoint.ServiceName][]endpoint.ServiceName

	connectedProxies     mapset.Set
	announcementChannels mapset.Set
}

// MeshCataloger is the mechanism by which the Service Mesh controller discovers all Envoy proxies connected to the catalog.
type MeshCataloger interface {
	// ListEndpoints constructs a map of service to weighted handlers with all endpoints the given Envoy proxy should be aware of.
	ListEndpoints(smi.ClientIdentity) (map[endpoint.ServiceName][]endpoint.WeightedService, error)

	// ListTrafficRoutes constructs a list of all the traffic policies /routes the given Envoy proxy should be aware of.
	ListTrafficRoutes(smi.ClientIdentity) ([]endpoint.TrafficTargetPolicies, error)

	// GetCertificateForService returns the SSL Certificate for the given service.
	// This certificate will be used for service-to-service mTLS.
	GetCertificateForService(endpoint.ServiceName) (certificate.Certificater, error)

	// RegisterProxy registers a newly connected proxy with the service mesh catalog.
	RegisterProxy(*envoy.Proxy)

	// UnregisterProxy unregisters an existing proxy from the service mesh catalog
	UnregisterProxy(*envoy.Proxy)

	// GetServicesByServiceAccountName returns a list of services corresponding to a service account, and refreshes the cache if requested
	GetServicesByServiceAccountName(endpoint.ServiceAccount, bool) []endpoint.ServiceName
}

type announcementChannel struct {
	announcer string
	channel   <-chan interface{}
}
