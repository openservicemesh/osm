package eds

import (
	"strings"

	xds_discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	"github.com/pkg/errors"

	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/endpoint"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/envoy/registry"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/service"
)

const (
	namespacedNameDelimiter = "/"
)

// NewResponse creates a new Endpoint Discovery Response.
func NewResponse(meshCatalog catalog.MeshCataloger, proxy *envoy.Proxy, request *xds_discovery.DiscoveryRequest, _ configurator.Configurator, _ certificate.Manager, _ *registry.ProxyRegistry) ([]types.Resource, error) {
	// If request comes through and requests specific endpoints, just attempt to answer those
	if request != nil && len(request.ResourceNames) > 0 {
		return fulfillEDSRequest(meshCatalog, proxy, request)
	}

	// Otherwise, generate all endpoint configuration for this proxy
	return generateEDSConfig(meshCatalog, proxy)
}

// fulfillEDSRequest replies only to requested EDS endpoints on Discovery Request
func fulfillEDSRequest(meshCatalog catalog.MeshCataloger, proxy *envoy.Proxy, request *xds_discovery.DiscoveryRequest) ([]types.Resource, error) {
	proxyIdentity, err := envoy.GetServiceAccountFromProxyCertificate(proxy.GetCertificateCommonName())
	if err != nil {
		log.Error().Err(err).Msgf("Error looking up identity for proxy %s", proxy.String())
		return nil, err
	}

	if request == nil {
		return nil, errors.Errorf("Endpoint discovery request for proxy %s cannot be nil", proxyIdentity)
	}

	var rdsResources []types.Resource
	for _, cluster := range request.ResourceNames {
		meshSvc, err := clusterToMeshSvc(cluster)
		if err != nil {
			log.Error().Err(err).Msgf("Error retrieving MeshService from Cluster %s", cluster)
			continue
		}
		endpoints, err := meshCatalog.ListAllowedEndpointsForService(proxyIdentity.ToServiceIdentity(), meshSvc)
		if err != nil {
			log.Error().Err(err).Msgf("Failed listing allowed endpoints for service %s, for proxy identity %s", meshSvc, proxyIdentity)
			continue
		}
		loadAssignment := newClusterLoadAssignment(meshSvc, endpoints)
		rdsResources = append(rdsResources, loadAssignment)
	}

	return rdsResources, nil
}

// generateEDSConfig generates all endpoints expected for a given proxy
func generateEDSConfig(meshCatalog catalog.MeshCataloger, proxy *envoy.Proxy) ([]types.Resource, error) {
	proxyIdentity, err := envoy.GetServiceAccountFromProxyCertificate(proxy.GetCertificateCommonName())
	if err != nil {
		log.Error().Err(err).Msgf("Error looking up identity for proxy %s", proxy.String())
		return nil, err
	}

	allowedEndpoints, err := getEndpointsForProxy(meshCatalog, proxyIdentity.ToServiceIdentity())
	if err != nil {
		log.Error().Err(err).Msgf("Error looking up endpoints for proxy %s", proxy.String())
		return nil, err
	}

	var rdsResources []types.Resource
	for svc, endpoints := range allowedEndpoints {
		loadAssignment := newClusterLoadAssignment(svc, endpoints)
		rdsResources = append(rdsResources, loadAssignment)
	}

	return rdsResources, nil
}

func clusterToMeshSvc(cluster string) (service.MeshService, error) {
	chunks := strings.Split(cluster, namespacedNameDelimiter)
	if len(chunks) != 2 {
		return service.MeshService{}, errors.Errorf("Invalid cluster name. Expected: <namespace>/<name>, Got: %s", cluster)
	}
	return service.MeshService{Namespace: chunks[0], Name: chunks[1]}, nil
}

// getEndpointsForProxy returns only those service endpoints that belong to the allowed outbound service accounts for the proxy
// Note: ServiceIdentity must be in the format "name.namespace" [https://github.com/openservicemesh/osm/issues/3188]
func getEndpointsForProxy(meshCatalog catalog.MeshCataloger, proxyIdentity identity.ServiceIdentity) (map[service.MeshService][]endpoint.Endpoint, error) {
	allowedServicesEndpoints := make(map[service.MeshService][]endpoint.Endpoint)

	for _, dstSvc := range meshCatalog.ListAllowedOutboundServicesForIdentity(proxyIdentity) {
		endpoints, err := meshCatalog.ListAllowedEndpointsForService(proxyIdentity, dstSvc)
		if err != nil {
			log.Error().Err(err).Msgf("Failed listing allowed endpoints for service %s for proxy identity %s", dstSvc, proxyIdentity)
			continue
		}
		allowedServicesEndpoints[dstSvc] = endpoints
	}
	log.Trace().Msgf("Allowed outbound service endpoints for proxy with identity %s: %v", proxyIdentity, allowedServicesEndpoints)
	return allowedServicesEndpoints, nil
}
