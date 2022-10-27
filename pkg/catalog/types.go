// Package catalog implements the MeshCataloger interface, which forms the central component in OSM that transforms
// outputs from all other components (SMI policies, Kubernetes services, endpoints etc.) into configuration that is
// consumed by the the proxy control plane component to program sidecar proxies.
// Reference: https://github.com/openservicemesh/osm/blob/main/DESIGN.md#5-mesh-catalog
package catalog

import (
	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/compute"
	"github.com/openservicemesh/osm/pkg/endpoint"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/logger"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/smi"
	"github.com/openservicemesh/osm/pkg/trafficpolicy"
	smiAccess "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/access/v1alpha3"
)

var (
	log = logger.New("mesh-catalog")
)

// MeshCatalog is the struct for the service catalog
type MeshCatalog struct {
	compute.Interface
	certManager *certificate.Manager
}

// MeshCataloger is the mechanism by which the Service Mesh controller discovers all Envoy proxies connected to the catalog.
type MeshCataloger interface {
	compute.Interface

	// ListOutboundServicesForIdentity list the services the given service identity is allowed to initiate outbound connections to
	ListOutboundServicesForIdentity(identity.ServiceIdentity) []service.MeshService

	// ListInboundServiceIdentities lists the downstream service identities that are allowed to connect to the given service identity
	ListInboundServiceIdentities(identity.ServiceIdentity) []identity.ServiceIdentity

	// ListOutboundServiceIdentities lists the upstream service identities the given service identity are allowed to connect to
	ListOutboundServiceIdentities(identity.ServiceIdentity) []identity.ServiceIdentity

	// ListAllowedUpstreamEndpointsForService returns the list of endpoints over which the downstream client identity
	// is allowed access the upstream service
	ListAllowedUpstreamEndpointsForService(identity.ServiceIdentity, service.MeshService) []endpoint.Endpoint

	// ListInboundTrafficTargetsWithRoutes returns a list traffic target objects composed of its routes for the given destination service identity
	ListInboundTrafficTargetsWithRoutes(identity.ServiceIdentity) ([]trafficpolicy.TrafficTargetWithRoutes, error)

	// GetOutboundMeshClusterConfigs returns the cluster configs for the outbound mesh traffic policy for the given downstream identity
	GetOutboundMeshClusterConfigs(identity.ServiceIdentity) []*trafficpolicy.MeshClusterConfig

	// GetOutboundMeshTrafficMatches returns the traffic matches for the outbound mesh traffic policy for the given downstream identity
	GetOutboundMeshTrafficMatches(identity.ServiceIdentity) []*trafficpolicy.TrafficMatch

	// GetOutboundMeshHTTPRouteConfigsPerPort returns a map of the given outbound traffic policy per port for the given downstream identity
	GetOutboundMeshHTTPRouteConfigsPerPort(identity.ServiceIdentity) map[int][]*trafficpolicy.OutboundTrafficPolicy

	// GetEgressClusterConfigs returns the cluster configs for the egress traffic policy associated with the given service identity.
	GetEgressClusterConfigs(identity.ServiceIdentity) ([]*trafficpolicy.EgressClusterConfig, error)

	// GetEgressTrafficMatches returns the traffic matches for the egress traffic policy associated with the given service identity.
	GetEgressTrafficMatches(identity.ServiceIdentity) ([]*trafficpolicy.TrafficMatch, error)

	// GetEgressHTTPRouteConfigsPerPort returns a map of the given egress http route config per port for the egress traffic policy associated with the given service identity.
	GetEgressHTTPRouteConfigsPerPort(identity.ServiceIdentity) map[int][]*trafficpolicy.EgressHTTPRouteConfig

	// GetIngressHTTPRoutePolicies returns the HTTP route policies for the ingress traffic policy for all mesh services
	GetIngressHTTPRoutePolicies([]service.MeshService) [][]*trafficpolicy.InboundTrafficPolicy

	// GetIngressHTTPRoutePoliciesForSvc returns the HTTP route policies for the ingress traffic policy for the given mesh service
	GetIngressHTTPRoutePoliciesForSvc(service.MeshService) []*trafficpolicy.InboundTrafficPolicy

	// GetIngressTrafficMatchesForSvc returns the traffic matches for the ingress traffic policy for the given mesh service
	GetIngressTrafficMatchesForSvc(service.MeshService) ([]*trafficpolicy.IngressTrafficMatch, error)

	// GetIngressHTTPRoutePolicies returns the ingress traffic matches for the ingress traffic policy for the given mesh service
	GetIngressTrafficMatches([]service.MeshService) [][]*trafficpolicy.IngressTrafficMatch

	ListServiceAccountsFromTrafficTargets() []identity.K8sServiceAccount

	// GetUpstreamServicesIncludeApex returns a list of all upstream services associated with the given list
	// of services. An upstream service is associated with another service if it is a backend for an apex/root service
	// in a TrafficSplit config. This function returns a list consisting of the given upstream services and all apex
	// services associated with each of those services.
	GetUpstreamServicesIncludeApex(upstreamServices []service.MeshService) []service.MeshService

	// ListTrafficTargetsByOptions returns a list of traffic targets that match the given options.
	ListTrafficTargetsByOptions(options ...smi.TrafficTargetListOption) []*smiAccess.TrafficTarget
}

type trafficDirection string

const (
	inbound  trafficDirection = "inbound"
	outbound trafficDirection = "outbound"
)
