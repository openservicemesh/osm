package eds

import (
	xds_discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"

	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/endpoint"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/envoy/memoize"
	"github.com/openservicemesh/osm/pkg/service"
)

// NewResponseMemoized creates a new Endpoint Discovery Response, but looks for it in a cache first.
func NewResponseMemoized(meshCatalog catalog.MeshCataloger, proxy *envoy.Proxy, _ *xds_discovery.DiscoveryRequest, _ configurator.Configurator, _ certificate.Manager) (*xds_discovery.DiscoveryResponse, error) {
	cacheKey, err := proxy.GetGroupID()
	if err != nil {
		log.Err(err).Msg("Error creating Memoization cache key; Using non-cached results")
		return NewResponse(meshCatalog, proxy, nil, nil, nil)
	}
	return memoize.Memoize(
		"EDS", cacheKey,
		NewResponse,
		meshCatalog, proxy, nil, nil, nil,
	)
}

// NewResponse creates a new Endpoint Discovery Response.
func NewResponse(meshCatalog catalog.MeshCataloger, proxy *envoy.Proxy, _ *xds_discovery.DiscoveryRequest, _ configurator.Configurator, _ certificate.Manager) (*xds_discovery.DiscoveryResponse, error) {
	proxyIdentity, err := certificate.GetServiceAccountFromProxyCertificate(proxy.GetCertificateCommonName())
	if err != nil {
		log.Error().Err(err).Msgf("Error looking up proxy identity for proxy with SerialNumber=%s on Pod with UID=%s", proxy.GetCertificateSerialNumber(), proxy.GetPodUID())
		return nil, err
	}

	allowedEndpoints, err := getEndpointsForProxy(meshCatalog, proxyIdentity)
	if err != nil {
		log.Error().Err(err).Msgf("Error looking up endpoints for proxy with SerialNumber=%s on Pod with UID=%s", proxy.GetCertificateSerialNumber(), proxy.GetPodUID())
		return nil, err
	}

	var protos []*any.Any
	for svc, endpoints := range allowedEndpoints {
		loadAssignment := newClusterLoadAssignment(svc, endpoints)
		proto, err := ptypes.MarshalAny(loadAssignment)
		if err != nil {
			log.Error().Err(err).Msgf("Error marshalling EDS payload for proxy with SerialNumber=%s on Pod with UID=%s", proxy.GetCertificateSerialNumber(), proxy.GetPodUID())
			continue
		}
		protos = append(protos, proto)
	}

	resp := &xds_discovery.DiscoveryResponse{
		Resources: protos,
		TypeUrl:   string(envoy.TypeEDS),
	}
	return resp, nil
}

// getEndpointsForProxy returns only those service endpoints that belong to the allowed outbound service accounts for the proxy
func getEndpointsForProxy(meshCatalog catalog.MeshCataloger, proxyIdentity service.K8sServiceAccount) (map[service.MeshService][]endpoint.Endpoint, error) {
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
