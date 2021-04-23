// Package catalog implements the MeshCataloger interface, which forms the central component in OSM that transforms
// outputs from all other components (SMI policies, Kubernetes services, endpoints etc.) into configuration that is
// consumed by the the proxy control plane component to program sidecar proxies.
// Reference: https://github.com/openservicemesh/osm/blob/main/DESIGN.md#5-mesh-catalog
package catalog

import (
	"github.com/google/uuid"
	"k8s.io/client-go/kubernetes"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/endpoint"
	"github.com/openservicemesh/osm/pkg/identity"
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

	// ListInboundTrafficPolicies returns all inbound traffic policies related to the given service identity and inbound services
	ListInboundTrafficPolicies(identity.ServiceIdentity, []service.MeshService) []*trafficpolicy.InboundTrafficPolicy

	// ListOutboundTrafficPolicies returns all outbound traffic policies related to the given service identity
	ListOutboundTrafficPolicies(identity.ServiceIdentity) []*trafficpolicy.OutboundTrafficPolicy

	// ListAllowedOutboundServicesForIdentity list the services the given service identity is allowed to initiate outbound connections to
	ListAllowedOutboundServicesForIdentity(identity.ServiceIdentity) []service.MeshService

	// ListAllowedInboundServiceIdentities lists the downstream service identities that can connect to the given service identity
	ListAllowedInboundServiceIdentities(identity.ServiceIdentity) ([]identity.ServiceIdentity, error)

	// ListAllowedOutboundServiceIdentities lists the upstream service identities the given service identity can connect to
	ListAllowedOutboundServiceIdentities(identity.ServiceIdentity) ([]identity.ServiceIdentity, error)

	// ListServiceIdentitiesForService lists the service identities associated with the given service
	ListServiceIdentitiesForService(service.MeshService) ([]identity.ServiceIdentity, error)

	// ListAllowedEndpointsForService returns the list of endpoints backing a service and its allowed service identities
	ListAllowedEndpointsForService(identity.ServiceIdentity, service.MeshService) ([]endpoint.Endpoint, error)

	// GetResolvableServiceEndpoints returns the resolvable set of endpoint over which a service is accessible using its FQDN.
	// These are the endpoint destinations we'd expect client applications sends the traffic towards to, when attempting to
	// reach a specific service.
	// If no LB/virtual IPs are assigned to the service, GetResolvableServiceEndpoints will return ListEndpointsForService
	GetResolvableServiceEndpoints(service.MeshService) ([]endpoint.Endpoint, error)

	// GetServicesFromEnvoyCertificate returns a list of services the given Envoy is a member of based on the certificate provided, which is a cert issued to an Envoy for XDS communication (not Envoy-to-Envoy).
	GetServicesFromEnvoyCertificate(certificate.CommonName) ([]service.MeshService, error)

	// GetIngressPoliciesForService returns the inbound traffic policies associated with an ingress service
	GetIngressPoliciesForService(service.MeshService) ([]*trafficpolicy.InboundTrafficPolicy, error)

	// GetTargetPortToProtocolMappingForService returns a mapping of the service's ports to their corresponding application protocol.
	// The ports returned are the actual ports on which the application exposes the service derived from the service's endpoints,
	// ie. 'spec.ports[].targetPort' instead of 'spec.ports[].port' for a Kubernetes service.
	GetTargetPortToProtocolMappingForService(service.MeshService) (map[uint32]string, error)

	// GetPortToProtocolMappingForService returns a mapping of the service's ports to their corresponding application protocol,
	// where the ports returned are the ones used by downstream clients in their requests. This can be different from the ports
	// actually exposed by the application binary, ie. 'spec.ports[].port' instead of 'spec.ports[].targetPort' for a Kubernetes service.
	GetPortToProtocolMappingForService(service.MeshService) (map[uint32]string, error)

	// ListInboundTrafficTargetsWithRoutes returns a list traffic target objects composed of its routes for the given destination service identity
	ListInboundTrafficTargetsWithRoutes(identity.ServiceIdentity) ([]trafficpolicy.TrafficTargetWithRoutes, error)
}

// certificateCommonNameMeta is the type that stores the metadata present in the CommonName field in a proxy's certificate
type certificateCommonNameMeta struct {
	ProxyUUID uuid.UUID
	// TODO(draychev): Change this to ServiceIdentity type (instead of string)
	ServiceAccount string
	Namespace      string
}

type trafficDirection string

const (
	inbound  trafficDirection = "inbound"
	outbound trafficDirection = "outbound"
)
