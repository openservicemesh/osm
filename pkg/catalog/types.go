package catalog

import (
	"sync"
	"time"

	"github.com/google/uuid"
	access "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/access/v1alpha3"
	spec "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/specs/v1alpha4"
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

	expectedProxies     sync.Map
	connectedProxies    sync.Map
	disconnectedProxies sync.Map

	// Current assumption is that OSM is working with a single Kubernetes cluster.
	// This is the API/REST interface to the cluster
	kubeClient kubernetes.Interface

	// This is the kubernetes client that operates async caches to avoid issuing synchronous
	// calls through kubeClient and instead relies on background cache synchronization and local
	// lookups
	kubeController k8s.Controller

	// Maintain a mapping of pod UID to CN of the Envoy on the given pod
	podUIDToCN sync.Map

	// Maintain a mapping of pod UID to certificate SerialNumber of the Envoy on the given pod
	podUIDToCertificateSerialNumber sync.Map
}

// MeshCataloger is the mechanism by which the Service Mesh controller discovers all Envoy proxies connected to the catalog.
type MeshCataloger interface {
	// GetSMISpec returns the SMI spec
	GetSMISpec() smi.MeshSpec

	// ListTrafficPolicies returns all the traffic policies for a given service that Envoy proxy should be aware of.
	ListTrafficPolicies(service.MeshService) ([]trafficpolicy.TrafficTarget, error)

	// ListTrafficPoliciesForServiceAccount returns all inbound and outbound traffic policies related to the given service account
	ListTrafficPoliciesForServiceAccount(service.K8sServiceAccount) ([]*trafficpolicy.InboundTrafficPolicy, []*trafficpolicy.OutboundTrafficPolicy, error)

	// ListAllowedInboundServices lists the inbound services allowed to connect to the given service.
	ListAllowedInboundServices(service.MeshService) ([]service.MeshService, error)

	// ListAllowedOutboundServicesForIdentity list the services the given service account is allowed to initiate outbound connections to
	ListAllowedOutboundServicesForIdentity(service.K8sServiceAccount) []service.MeshService

	// ListAllowedInboundServiceAccounts lists the downstream service accounts that can connect to the given service account
	ListAllowedInboundServiceAccounts(service.K8sServiceAccount) ([]service.K8sServiceAccount, error)

	// ListAllowedOutboundServiceAccounts lists the upstream service accounts the given service account can connect to
	ListAllowedOutboundServiceAccounts(service.K8sServiceAccount) ([]service.K8sServiceAccount, error)

	// ListServiceAccountsForService lists the service accounts associated with the given service
	ListServiceAccountsForService(service.MeshService) ([]service.K8sServiceAccount, error)

	// ListSMIPolicies lists SMI policies.
	ListSMIPolicies() ([]*split.TrafficSplit, []service.WeightedService, []service.K8sServiceAccount, []*spec.HTTPRouteGroup, []*access.TrafficTarget)

	// ListEndpointsForService returns the list of individual instance endpoint backing a service
	ListEndpointsForService(service.MeshService) ([]endpoint.Endpoint, error)

	// GetResolvableServiceEndpoints returns the resolvable set of endpoint over which a service is accessible using its FQDN.
	// These are the endpoint destinations we'd expect client applications sends the traffic towards to, when attempting to
	// reach a specific service.
	// If no LB/virtual IPs are assigned to the service, GetResolvableServiceEndpoints will return ListEndpointsForService
	GetResolvableServiceEndpoints(service.MeshService) ([]endpoint.Endpoint, error)

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

	// GetResolvableHostnamesForUpstreamService returns the hostnames over which an upstream service is accessible from a downstream service
	GetResolvableHostnamesForUpstreamService(downstream, upstream service.MeshService) ([]string, error)

	//GetWeightedClusterForService returns the weighted cluster for a service
	GetWeightedClusterForService(service service.MeshService) (service.WeightedCluster, error)

	// GetIngressRoutesPerHost returns the HTTP route matches per host associated with an ingress service
	GetIngressRoutesPerHost(service.MeshService) (map[string][]trafficpolicy.HTTPRouteMatch, error)

	// ListMonitoredNamespaces lists namespaces monitored by the control plane
	ListMonitoredNamespaces() []string

	// GetTargetPortToProtocolMappingForService returns a mapping of the service's ports to their corresponding application protocol.
	// The ports returned are the actual ports on which the application exposes the service derived from the service's endpoints,
	// ie. 'spec.ports[].targetPort' instead of 'spec.ports[].port' for a Kubernetes service.
	GetTargetPortToProtocolMappingForService(service.MeshService) (map[uint32]string, error)

	// GetTargetPortToProtocolMappingForService returns a mapping of the service's ports to their corresponding application protocol,
	// where the ports returned are the ones used by downstream clients in their requests. This can be different from the ports
	// actually exposed by the application binary, ie. 'spec.ports[].port' instead of 'spec.ports[].targetPort' for a Kubernetes service.
	GetPortToProtocolMappingForService(service.MeshService) (map[uint32]string, error)

	// ListInboundTrafficTargetsWithRoutes returns a list traffic target objects composed of its routes for the given destination service account
	ListInboundTrafficTargetsWithRoutes(service.K8sServiceAccount) ([]trafficpolicy.TrafficTargetWithRoutes, error)
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
	ProxyUUID      uuid.UUID
	ServiceAccount string
	Namespace      string
}

type trafficDirection string

const (
	inbound  trafficDirection = "inbound"
	outbound trafficDirection = "outbound"
)
