package catalog

import (
	"sync"

	mapset "github.com/deckarep/golang-set"

	"github.com/open-service-mesh/osm/pkg/certificate"
	"github.com/open-service-mesh/osm/pkg/endpoint"
	"github.com/open-service-mesh/osm/pkg/envoy"
	"github.com/open-service-mesh/osm/pkg/logger"
	"github.com/open-service-mesh/osm/pkg/smi"
)

var (
	log = logger.New("mesh-catalog")
)

// MeshCatalog is the struct for the service catalog
type MeshCatalog struct {
	endpointsProviders []endpoint.Provider
	meshSpec           smi.MeshSpec
	certManager        certificate.Manager

	servicesCache        map[endpoint.WeightedService][]endpoint.Endpoint
	servicesMutex        sync.Mutex
	certificateCache     map[endpoint.NamespacedService]certificate.Certificater
	serviceAccountsCache map[endpoint.NamespacedServiceAccount][]endpoint.NamespacedService

	connectedProxies     mapset.Set
	announcementChannels mapset.Set
}

// MeshCataloger is the mechanism by which the Service Mesh controller discovers all Envoy proxies connected to the catalog.
type MeshCataloger interface {
	// ListTrafficSplitEndpoints constructs a map of service to weighted handlers with all endpoints the given Envoy proxy should be aware of.
	ListTrafficSplitEndpoints(endpoint.NamespacedService) ([]endpoint.WeightedServiceEndpoints, error)

	// ListTrafficPolicies returns all the traffic policies for a given service that Envoy proxy should be aware of.
	ListTrafficPolicies(endpoint.NamespacedService) ([]endpoint.TrafficTargetPolicies, error)

	// GetCertificateForService returns the SSL Certificate for the given service.
	// This certificate will be used for service-to-service mTLS.
	GetCertificateForService(endpoint.NamespacedService) (certificate.Certificater, error)

	// RegisterProxy registers a newly connected proxy with the service mesh catalog.
	RegisterProxy(*envoy.Proxy)

	// UnregisterProxy unregisters an existing proxy from the service mesh catalog
	UnregisterProxy(*envoy.Proxy)

	// GetServicesByServiceAccountName returns a list of services corresponding to a service account, and refreshes the cache if requested
	GetServicesByServiceAccountName(endpoint.NamespacedServiceAccount, bool) []endpoint.NamespacedService
}

type announcementChannel struct {
	announcer string
	channel   <-chan interface{}
}
