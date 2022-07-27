package eds

import (
	"fmt"
	"strconv"
	"strings"

	xds_discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"

	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/endpoint"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/envoy/handler"
	"github.com/openservicemesh/osm/pkg/envoy/registry"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/service"
)

// fulfillEDSRequest replies only to requested EDS endpoints on Discovery Request
func fulfillEDSRequest(meshCatalog catalog.MeshCataloger, proxy *envoy.Proxy, request *xds_discovery.DiscoveryRequest) ([]types.Resource, error) {
	if request == nil {
		return nil, fmt.Errorf("Endpoint discovery request for proxy %s cannot be nil", proxy.Identity)
	}

	var rdsResources []types.Resource
	for _, cluster := range request.ResourceNames {
		meshSvc, err := clusterToMeshSvc(cluster)
		if err != nil {
			log.Error().Err(err).Msgf("Error retrieving MeshService from Cluster %s", cluster)
			continue
		}
		endpoints := meshCatalog.ListAllowedUpstreamEndpointsForService(proxy.Identity, meshSvc)
		log.Trace().Msgf("Endpoints for upstream cluster %s for downstream proxy identity %s: %v", cluster, proxy.Identity, endpoints)
		loadAssignment := newClusterLoadAssignment(meshSvc, endpoints)
		rdsResources = append(rdsResources, loadAssignment)
	}

	return rdsResources, nil
}

type Handler struct {
	handler.XDSHandler

	MeshCatalog      catalog.MeshCataloger
	Proxy            *envoy.Proxy
	DiscoveryRequest *xds_discovery.DiscoveryRequest
	Cfg              configurator.Configurator
	CertManager      *certificate.Manager
	ProxyRegistry    *registry.ProxyRegistry
}

func (h *Handler) Respond() ([]types.Resource, error) {
	// If request comes through and requests specific endpoints, just attempt to answer those
	if h.DiscoveryRequest != nil && len(h.DiscoveryRequest.ResourceNames) > 0 {
		return fulfillEDSRequest(h.MeshCatalog, h.Proxy, h.DiscoveryRequest)
	}

	// Otherwise, generate all endpoint configuration for this proxy
	var edsResources []types.Resource
	upstreamSvcEndpoints := getUpstreamEndpointsForProxyIdentity(h.MeshCatalog, h.Proxy.Identity)

	for svc, endpoints := range upstreamSvcEndpoints {
		loadAssignment := newClusterLoadAssignment(svc, endpoints)
		edsResources = append(edsResources, loadAssignment)
	}

	return edsResources, nil
}

// clusterToMeshSvc returns the MeshService associated with the given cluster name
func clusterToMeshSvc(cluster string) (service.MeshService, error) {
	splitFunc := func(r rune) bool {
		return r == '/' || r == '|'
	}

	chunks := strings.FieldsFunc(cluster, splitFunc)
	if len(chunks) != 3 {
		return service.MeshService{}, fmt.Errorf("Invalid cluster name. Expected: <namespace>/<name>|<port>, got: %s", cluster)
	}

	port, err := strconv.ParseUint(chunks[2], 10, 16)
	if err != nil {
		return service.MeshService{}, fmt.Errorf("Invalid cluster port %s, expected int value: %w", chunks[2], err)
	}

	return service.MeshService{
		Namespace: chunks[0],
		Name:      chunks[1],

		// The port always maps to MeshService.TargetPort and not MeshService.Port because
		// endpoints of a service are derived from it's TargetPort and not Port.
		TargetPort: uint16(port),
	}, nil
}

// getUpstreamEndpointsForProxyIdentity returns only those service endpoints that belong to the allowed upstream service accounts for the proxy
// Note: ServiceIdentity must be in the format "name.namespace" [https://github.com/openservicemesh/osm/issues/3188]
func getUpstreamEndpointsForProxyIdentity(meshCatalog catalog.MeshCataloger, proxyIdentity identity.ServiceIdentity) map[service.MeshService][]endpoint.Endpoint {
	allowedServicesEndpoints := make(map[service.MeshService][]endpoint.Endpoint)

	for _, dstSvc := range meshCatalog.ListOutboundServicesForIdentity(proxyIdentity) {
		allowedServicesEndpoints[dstSvc] = meshCatalog.ListAllowedUpstreamEndpointsForService(proxyIdentity, dstSvc)
	}

	log.Trace().Msgf("Allowed outbound service endpoints for proxy with identity %s: %v", proxyIdentity, allowedServicesEndpoints)
	return allowedServicesEndpoints
}

func (h *Handler) SetMeshCataloger(cataloger catalog.MeshCataloger) {
	h.MeshCatalog = cataloger
}

func (h *Handler) SetProxy(proxy *envoy.Proxy) {
	h.Proxy = proxy
}

func (h *Handler) SetDiscoveryRequest(request *xds_discovery.DiscoveryRequest) {
	h.DiscoveryRequest = request
}

func (h *Handler) SetConfigurator(cfg configurator.Configurator) {
	h.Cfg = cfg
}

func (h *Handler) SetCertManager(certManager *certificate.Manager) {
	h.CertManager = certManager
}

func (h *Handler) SetProxyRegistry(proxyRegistry *registry.ProxyRegistry) {
	h.ProxyRegistry = proxyRegistry
}
