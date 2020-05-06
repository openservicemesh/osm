package catalog

import (
	"sync"
	"time"

	"k8s.io/client-go/kubernetes"

	mapset "github.com/deckarep/golang-set"

	"github.com/open-service-mesh/osm/pkg/certificate"
	"github.com/open-service-mesh/osm/pkg/endpoint"
	"github.com/open-service-mesh/osm/pkg/envoy"
	"github.com/open-service-mesh/osm/pkg/ingress"
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
	ingressMonitor     ingress.Monitor

	servicesCache                 map[endpoint.WeightedService][]endpoint.Endpoint
	servicesMutex                 sync.Mutex
	certificateCache              map[endpoint.NamespacedService]certificate.Certificater
	serviceAccountToServicesCache map[endpoint.NamespacedServiceAccount][]endpoint.NamespacedService

	expectedProxies      map[certificate.CommonName]expectedProxy
	connectedProxies     map[certificate.CommonName]connectedProxy
	announcementChannels mapset.Set

	// Current assumption is that OSM is working with a single Kubernetes cluster.
	// This here is the client to that cluster.
	kubeClient kubernetes.Interface
}

// MeshCataloger is the mechanism by which the Service Mesh controller discovers all Envoy proxies connected to the catalog.
type MeshCataloger interface {
	// ListTrafficSplitEndpoints constructs a map of service to weighted handlers with all endpoints the given Envoy proxy should be aware of.
	ListTrafficSplitEndpoints(endpoint.NamespacedService) ([]endpoint.WeightedServiceEndpoints, error)

	// ListTrafficPolicies returns all the traffic policies for a given service that Envoy proxy should be aware of.
	ListTrafficPolicies(endpoint.NamespacedService) ([]endpoint.TrafficPolicy, error)

	// ListEndpointsForService returns the list of provider endpoints corresponding to a service
	ListEndpointsForService(endpoint.ServiceName) ([]endpoint.Endpoint, error)

	// GetCertificateForService returns the SSL Certificate for the given service.
	// This certificate will be used for service-to-service mTLS.
	GetCertificateForService(endpoint.NamespacedService) (certificate.Certificater, error)

	// ExpectProxy catalogs the fact that a certificate was issued for an Envoy proxy and this is expected to connect to XDS.
	ExpectProxy(certificate.CommonName)

	// GetServiceFromEnvoyCertificate returns the single service given Envoy is a member of based on the certificate provided, which is a cert issued to an Envoy for XDS communication (not Envoy-to-Envoy).
	GetServiceFromEnvoyCertificate(certificate.CommonName) (*endpoint.NamespacedService, error)

	// RegisterProxy registers a newly connected proxy with the service mesh catalog.
	RegisterProxy(*envoy.Proxy)

	// UnregisterProxy unregisters an existing proxy from the service mesh catalog
	UnregisterProxy(*envoy.Proxy)

	// GetServicesByServiceAccountName returns a list of services corresponding to a service account, and refreshes the cache if requested
	GetServicesByServiceAccountName(endpoint.NamespacedServiceAccount, bool) []endpoint.NamespacedService

	//GetDomainForService returns the domain name of a service
	GetDomainForService(service endpoint.NamespacedService) (string, error)

	//GetWeightedClusterForService returns the weighted cluster for a service
	GetWeightedClusterForService(service endpoint.NamespacedService) (endpoint.WeightedCluster, error)

	// IsIngressService returns a boolean indicating if the service is a backend for an ingress resource
	IsIngressService(endpoint.NamespacedService) (bool, error)

	// GetIngressRoutePoliciesPerDomain returns the route policies per domain associated with an ingress service
	GetIngressRoutePoliciesPerDomain(endpoint.NamespacedService) (map[string][]endpoint.RoutePolicy, error)

	// GetIngressWeightedCluster returns the weighted cluster for an ingress service
	GetIngressWeightedCluster(endpoint.NamespacedService) (endpoint.WeightedCluster, error)
}

type announcementChannel struct {
	announcer string
	channel   <-chan interface{}
}

type expectedProxy struct {
	// The time the certificate, identified by CN, for the expected proxy was issued on
	issuedOn time.Time
}
type connectedProxy struct {
	// Proxy which connected to the XDS control plane
	proxy *envoy.Proxy

	// When the proxy connected to the XDS control plane
	connectedOn time.Time
}

// certificateCommonNameMeta is the type that stores the metadata present in the CommonName field in a proxy's certificate
type certificateCommonNameMeta struct {
	ProxyID   string
	Namespace string
}
