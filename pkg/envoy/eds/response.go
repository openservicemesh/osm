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
	"github.com/openservicemesh/osm/pkg/envoy/registry"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/service"
)

// NewResponse creates a new Endpoint Discovery Response.
func NewResponse(meshCatalog catalog.MeshCataloger, proxy *envoy.Proxy, request *xds_discovery.DiscoveryRequest, _ configurator.Configurator, _ *certificate.Manager, _ *registry.ProxyRegistry) ([]types.Resource, error) {
	// If request comes through and requests specific endpoints, just attempt to answer those
	if request != nil && len(request.ResourceNames) > 0 {
		return fulfillEDSRequest(meshCatalog, proxy, request)
	}

	// Otherwise, generate all endpoint configuration for this proxy
	return generateEDSConfig(meshCatalog, proxy)
}

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

// generateEDSConfig generates all endpoints expected for a given proxy
func generateEDSConfig(meshCatalog catalog.MeshCataloger, proxy *envoy.Proxy) ([]types.Resource, error) {
	var edsResources []types.Resource
	upstreamSvcEndpoints := getUpstreamEndpointsForProxyIdentity(meshCatalog, proxy.Identity)

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
