package rds

import (
	xds_discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/golang/protobuf/ptypes"

	"github.com/openservicemesh/osm/pkg/catalog"
	cat "github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/envoy/route"
	"github.com/openservicemesh/osm/pkg/trafficpolicy"
)

func newResponse(catalog catalog.MeshCataloger, proxy *envoy.Proxy) (*xds_discovery.DiscoveryResponse, error) {
	proxyIdentity, err := cat.GetServiceAccountFromProxyCertificate(proxy.GetCertificateCommonName())
	if err != nil {
		log.Error().Err(err).Msgf("Error looking up Service Account for Envoy with serial number=%q", proxy.GetCertificateSerialNumber())
		return nil, err
	}
	inboundTrafficPolicies, outboundTrafficPolicies, err := catalog.ListTrafficPoliciesForServiceAccount(proxyIdentity)
	if err != nil {
		return nil, err
	}

	// Get Ingress inbound policies for the proxy
	services, err := catalog.GetServicesFromEnvoyCertificate(proxy.GetCertificateCommonName())
	if err != nil {
		log.Error().Err(err).Msgf("Error looking up services for Envoy with serial number=%q", proxy.GetCertificateSerialNumber())
		return nil, err
	}
	for _, svc := range services {
		ingressInboundPolicies, err := catalog.GetIngressPoliciesForService(svc, proxyIdentity)
		if err != nil {
			log.Error().Err(err).Msgf("Error looking up ingress policies for service=%s", svc.String())
			return nil, err
		}
		inboundTrafficPolicies = trafficpolicy.MergeInboundPolicies(true, inboundTrafficPolicies, ingressInboundPolicies...)
	}

	routeConfiguration := route.BuildRouteConfiguration(inboundTrafficPolicies, outboundTrafficPolicies)
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
