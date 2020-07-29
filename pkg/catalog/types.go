package catalog

import (
	"sync"
	"time"

	mapset "github.com/deckarep/golang-set"
	target "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/access/v1alpha1"
	spec "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/specs/v1alpha2"
	split "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/split/v1alpha2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/endpoint"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/ingress"
	"github.com/openservicemesh/osm/pkg/logger"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/smi"
	"github.com/openservicemesh/osm/pkg/trafficpolicy"
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
	configurator       configurator.Configurator

	expectedProxies     map[certificate.CommonName]expectedProxy
	expectedProxiesLock sync.Mutex

	connectedProxies     map[certificate.CommonName]connectedProxy
	connectedProxiesLock sync.Mutex

	disconnectedProxies     map[certificate.CommonName]disconnectedProxy
	disconnectedProxiesLock sync.Mutex

	announcementChannels mapset.Set

	// Current assumption is that OSM is working with a single Kubernetes cluster.
	// This here is the client to that cluster.
	kubeClient kubernetes.Interface
}

// MeshCataloger is the mechanism by which the Service Mesh controller discovers all Envoy proxies connected to the catalog.
type MeshCataloger interface {
	// GetSMISpec returns the SMI spec
	GetSMISpec() smi.MeshSpec

	// ListTrafficPolicies returns all the traffic policies for a given service that Envoy proxy should be aware of.
	ListTrafficPolicies(service.NamespacedService) ([]trafficpolicy.TrafficTarget, error)

	// ListAllowedInboundServices lists the inbound services allowed to connect to the given service.
	ListAllowedInboundServices(service.NamespacedService) ([]service.NamespacedService, error)

	// ListAllowedOutboundServices lists the services the given service is allowed outbound connections to.
	ListAllowedOutboundServices(service.NamespacedService) ([]service.NamespacedService, error)

	// ListSMIPolicies lists SMI policies.
	ListSMIPolicies() ([]*split.TrafficSplit, []service.WeightedService, []service.K8sServiceAccount, []*spec.HTTPRouteGroup, []*target.TrafficTarget, []*corev1.Service)

	// ListEndpointsForService returns the list of provider endpoints corresponding to a service
	ListEndpointsForService(service.Name) ([]endpoint.Endpoint, error)

	// GetCertificateForService returns the SSL Certificate for the given service.
	// This certificate will be used for service-to-service mTLS.
	GetCertificateForService(service.NamespacedService) (certificate.Certificater, error)

	// ExpectProxy catalogs the fact that a certificate was issued for an Envoy proxy and this is expected to connect to XDS.
	ExpectProxy(certificate.CommonName)

	// GetServiceFromEnvoyCertificate returns the single service given Envoy is a member of based on the certificate provided, which is a cert issued to an Envoy for XDS communication (not Envoy-to-Envoy).
	GetServiceFromEnvoyCertificate(certificate.CommonName) (*service.NamespacedService, error)

	// RegisterProxy registers a newly connected proxy with the service mesh catalog.
	RegisterProxy(*envoy.Proxy)

	// UnregisterProxy unregisters an existing proxy from the service mesh catalog
	UnregisterProxy(*envoy.Proxy)

	// GetServiceForServiceAccount returns the service corresponding to a service account
	GetServiceForServiceAccount(service.K8sServiceAccount) (service.NamespacedService, error)

	//GetDomainForService returns the domain name of a service
	GetDomainForService(service service.NamespacedService, routeHeaders map[string]string) (string, error)

	//GetWeightedClusterForService returns the weighted cluster for a service
	GetWeightedClusterForService(service service.NamespacedService) (service.WeightedCluster, error)

	// GetIngressRoutesPerHost returns the routes per host associated with an ingress service
	GetIngressRoutesPerHost(service.NamespacedService) (map[string][]trafficpolicy.Route, error)
}

type announcementChannel struct {
	announcer string
	channel   <-chan interface{}
}

type expectedProxy struct {
	// The time the certificate, identified by CN, for the expected proxy was issued on
	certificateIssuedAt time.Time
}

type connectedProxy struct {
	// Proxy which connected to the XDS control plane
	proxy *envoy.Proxy

	// When the proxy connected to the XDS control plane
	connectedAt time.Time
}

type disconnectedProxy struct {
	lastSeen time.Time
}

// certificateCommonNameMeta is the type that stores the metadata present in the CommonName field in a proxy's certificate
type certificateCommonNameMeta struct {
	ProxyID        string
	ServiceAccount string
	Namespace      string
}

type direction string

const (
	inbound  direction = "inbound"
	outbound direction = "outbound"
)
