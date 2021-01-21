package rds

import (
	xds_discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/golang/protobuf/ptypes"

	"github.com/openservicemesh/osm/pkg/catalog"
	cat "github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/envoy/route"
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

	resp := &xds_discovery.DiscoveryResponse{
		TypeUrl: string(envoy.TypeRDS),
	}

	// TODO merge ingress policies with existing inboundTrafficPolicies (issue #2367)
	routeConfiguration := route.BuildRouteConfiguration(inboundTrafficPolicies, outboundTrafficPolicies)

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
