package rds

import (
	xds_discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/golang/protobuf/ptypes"

	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/envoy/memoize"
	"github.com/openservicemesh/osm/pkg/envoy/route"
	"github.com/openservicemesh/osm/pkg/trafficpolicy"
)

// NewResponseMemoized creates a new Route Discovery Response.
func NewResponseMemoized(meshCatalog catalog.MeshCataloger, proxy *envoy.Proxy, _ *xds_discovery.DiscoveryRequest, _ configurator.Configurator, _ certificate.Manager) (*xds_discovery.DiscoveryResponse, error) {
	cacheKey, err := proxy.GetGroupID()
	if err != nil {
		log.Err(err).Msg("Error creating Memoization cache key; Using non-cached results")
		return NewResponse(meshCatalog, proxy, nil, nil, nil)
	}
	return memoize.Memoize(
		"RDS", cacheKey,
		NewResponse,
		meshCatalog, proxy, nil, nil, nil,
	)
}

// NewResponse creates a new Route Discovery Response.
func NewResponse(meshCatalog catalog.MeshCataloger, proxy *envoy.Proxy, _ *xds_discovery.DiscoveryRequest, _ configurator.Configurator, _ certificate.Manager) (*xds_discovery.DiscoveryResponse, error) {
	var inboundTrafficPolicies []*trafficpolicy.InboundTrafficPolicy
	var outboundTrafficPolicies []*trafficpolicy.OutboundTrafficPolicy

	proxyIdentity, err := certificate.GetServiceAccountFromProxyCertificate(proxy.GetCertificateCommonName())
	if err != nil {
		log.Error().Err(err).Msgf("Error looking up Service Account for Envoy with serial number=%q", proxy.GetCertificateSerialNumber())
		return nil, err
	}

	services, err := meshCatalog.GetServicesFromEnvoyCertificate(proxy.GetCertificateCommonName())
	if err != nil {
		log.Error().Err(err).Msgf("Error looking up services for Envoy with serial number=%q", proxy.GetCertificateSerialNumber())
		return nil, err
	}

	// Build traffic policies from  either SMI Traffic Target and Traffic Split or service discovery
	// depending on whether permissive mode is enabled or not
	inboundTrafficPolicies = meshCatalog.ListInboundTrafficPolicies(proxyIdentity, services)
	outboundTrafficPolicies = meshCatalog.ListOutboundTrafficPolicies(proxyIdentity)

	// Get Ingress inbound policies for the proxy
	for _, svc := range services {
		ingressInboundPolicies, err := meshCatalog.GetIngressPoliciesForService(svc)
		if err != nil {
			log.Error().Err(err).Msgf("Error looking up ingress policies for service=%s", svc.String())
			return nil, err
		}
		inboundTrafficPolicies = trafficpolicy.MergeInboundPolicies(true, inboundTrafficPolicies, ingressInboundPolicies...)
	}

	routeConfiguration := route.BuildRouteConfiguration(inboundTrafficPolicies, outboundTrafficPolicies, proxy)
	resp := &xds_discovery.DiscoveryResponse{
		TypeUrl: string(envoy.TypeRDS),
	}

	for _, config := range routeConfiguration {
		marshalledRouteConfig, err := ptypes.MarshalAny(config)
		if err != nil {
			log.Error().Err(err).Msgf("Failed to marshal route config for proxy")
			return nil, err
		}
		resp.Resources = append(resp.Resources, marshalledRouteConfig)
	}

	return resp, nil
}
