package rds

import (
	xds_route "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	xds_discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/golang/protobuf/ptypes"

	cat "github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/envoy/route"
)

// NewResponse creates a new Route Discovery Response.
func NewResponse(catalog cat.MeshCataloger, proxy *envoy.Proxy, _ *xds_discovery.DiscoveryRequest, _ configurator.Configurator, _ certificate.Manager) (*xds_discovery.DiscoveryResponse, error) {
	//svcList, err := catalog.GetServicesFromEnvoyCertificate(proxy.GetCommonName())
	proxyIdentity, err := cat.GetServiceAccountFromProxyCertificate(proxy.GetCommonName())
	if err != nil {
		log.Error().Err(err).Msgf("Error looking up Service Account for Envoy with CN=%q", proxy.GetCommonName())
		return nil, err
	}
	// Github Issue #1575
	log.Debug().Msgf("Proxy Service Account %#v", proxyIdentity)
	//proxyServiceName := svcList[0]

	inboundTrafficPolicies, outboundTrafficPolicies, _ := catalog.ListTrafficPoliciesForService(proxyIdentity)

	resp := &xds_discovery.DiscoveryResponse{
		TypeUrl: string(envoy.TypeRDS),
	}

	var routeConfiguration []*xds_route.RouteConfiguration

	// fetch ingress routes
	// merge ingress routes on top of existing
	/* TODO
	if err = updateRoutesForIngress(proxyServiceName, catalog, inboundAggregatedRoutesByHostnames); err != nil {
		return nil, err
	}
	*/

	routeConfiguration = route.BuildRouteConfiguration(inboundTrafficPolicies, outboundTrafficPolicies)
	log.Debug().Msgf("routeConfiguration(proxyServiceName %#v): %+v", proxyIdentity, routeConfiguration)

	for _, config := range routeConfiguration {
		marshalledRouteConfig, err := ptypes.MarshalAny(config)
		if err != nil {
			log.Error().Err(err).Msgf("Failed to marshal route config for proxy")
			return nil, err
		}
		resp.Resources = append(resp.Resources, marshalledRouteConfig)

	}
	log.Debug().Msgf("proxyServiceName %#v, LEN %v resp.Resources: %+v \n ", proxyIdentity, len(resp.Resources), resp.Resources)

	return resp, nil
}
