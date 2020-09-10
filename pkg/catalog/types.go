package catalog

import (
	"sync"
	"time"

	mapset "github.com/deckarep/golang-set"
	target "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/access/v1alpha2"
	spec "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/specs/v1alpha3"
	split "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/split/v1alpha2"
	"k8s.io/client-go/kubernetes"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/endpoint"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/ingress"
	k8s "github.com/openservicemesh/osm/pkg/kubernetes"
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
	// This is the API/REST interface to the cluster
	kubeClient kubernetes.Interface

	// This is the kubernetes client that operates async caches to avoid issuing synchronous
	// calls through kubeClient and instead relies on background cache synchronization and local
	// lookups
	kubeController k8s.Controller
}

// MeshCataloger is the mechanism by which the Service Mesh controller discovers all Envoy proxies connected to the catalog.
type MeshCataloger interface {
	// GetSMISpec returns the SMI spec
	GetSMISpec() smi.MeshSpec

	// ListTrafficPolicies returns all the traffic policies for a given service that Envoy proxy should be aware of.
	ListTrafficPolicies(service.MeshService) ([]trafficpolicy.TrafficTarget, error)

	// ListAllowedInboundServices lists the inbound services allowed to connect to the given service.
	ListAllowedInboundServices(service.MeshService) ([]service.MeshService, error)

	// ListAllowedOutboundServices lists the services the given service is allowed outbound connections to.
	ListAllowedOutboundServices(service.MeshService) ([]service.MeshService, error)

	// ListSMIPolicies lists SMI policies.
	ListSMIPolicies() ([]*split.TrafficSplit, []service.WeightedService, []service.K8sServiceAccount, []*spec.HTTPRouteGroup, []*target.TrafficTarget)

	// ListEndpointsForService returns the list of individual instance endpoint backing a service
	ListEndpointsForService(service.MeshService) ([]endpoint.Endpoint, error)

	// GetResolvableServiceEndpoints returns the resolvable set of endpoint over which a service is accessible using its FQDN.
	// These are the endpoint destinations we'd expect client applications sends the traffic towards to, when attemtpting to
	// reach a specific service.
	// If no LB/virtual IPs are assigned to the service, GetResolvableServiceEndpoints will return ListEndpointsForService
	GetResolvableServiceEndpoints(service.MeshService) ([]endpoint.Endpoint, error)

	// GetCertificateForService returns the SSL Certificate for the given service.
	// This certificate will be used for service-to-service mTLS.
	GetCertificateForService(service.MeshService) (certificate.Certificater, error)

	// ExpectProxy catalogs the fact that a certificate was issued for an Envoy proxy and this is expected to connect to XDS.
	ExpectProxy(certificate.CommonName)

	// GetServicesFromEnvoyCertificate returns a list of services the given Envoy is a member of based on the certificate provided, which is a cert issued to an Envoy for XDS communication (not Envoy-to-Envoy).
	GetServicesFromEnvoyCertificate(certificate.CommonName) ([]service.MeshService, error)

	// RegisterProxy registers a newly connected proxy with the service mesh catalog.
	RegisterProxy(*envoy.Proxy)

	// UnregisterProxy unregisters an existing proxy from the service mesh catalog
	UnregisterProxy(*envoy.Proxy)

	// GetServicesForServiceAccount returns a list of services corresponding to a service account
	GetServicesForServiceAccount(service.K8sServiceAccount) ([]service.MeshService, error)

	// GetHostnamesForService returns the hostnames for a service
	// TODO(ref: PR #1316): return a list of strings
	GetHostnamesForService(service service.MeshService) (string, error)

	//GetWeightedClusterForService returns the weighted cluster for a service
	GetWeightedClusterForService(service service.MeshService) (service.WeightedCluster, error)

	// GetIngressRoutesPerHost returns the HTTP routes per host associated with an ingress service
	GetIngressRoutesPerHost(service.MeshService) (map[string][]trafficpolicy.HTTPRoute, error)

	// ListMonitoredNamespaces lists namespaces monitored by the control plane
	ListMonitoredNamespaces() []string
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
