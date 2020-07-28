package lds

import (
	envoy_api_v2_core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	envoy_api_v2_listener "github.com/envoyproxy/go-control-plane/envoy/api/v2/listener"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"github.com/golang/protobuf/ptypes"

	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/envoy/route"
	"github.com/openservicemesh/osm/pkg/service"
)

func getInboundInMeshFilterChain(proxyServiceName service.NamespacedService, cfg configurator.Configurator) (*envoy_api_v2_listener.FilterChain, error) {
	marshalledDownstreamTLSContext, err := envoy.MessageToAny(envoy.GetDownstreamTLSContext(proxyServiceName, true /* mTLS */))
	if err != nil {
		log.Error().Err(err).Msgf("Error marshalling DownstreamTLSContext object for proxy %s", proxyServiceName)
		return nil, err
	}

	inboundConnManager := getHTTPConnectionManager(route.InboundRouteConfigName, cfg)
	marshalledInboundConnManager, err := ptypes.MarshalAny(inboundConnManager)
	if err != nil {
		log.Error().Err(err).Msgf("Error marshalling inbound HttpConnectionManager object for proxy %s", proxyServiceName)
		return nil, err
	}

	filterChain := &envoy_api_v2_listener.FilterChain{
		Filters: []*envoy_api_v2_listener.Filter{
			{
				Name: wellknown.HTTPConnectionManager,
				ConfigType: &envoy_api_v2_listener.Filter_TypedConfig{
					TypedConfig: marshalledInboundConnManager,
				},
			},
		},

		// Apply this filter chain only to requests where the auth.UpstreamTlsContext.Sni matches
		// one from the list of ServerNames provided below.
		// This field is configured by the GetDownstreamTLSContext() function.
		// This is not a field obtained from the mTLS Certificate.
		FilterChainMatch: &envoy_api_v2_listener.FilterChainMatch{
			ServerNames:          []string{proxyServiceName.GetCommonName().String()},
			TransportProtocol:    envoy.TransportProtocolTLS,
			ApplicationProtocols: envoy.ALPNInMesh, // in-mesh proxies will advertise this, set in UpstreamTlsContext
		},

		TransportSocket: &envoy_api_v2_core.TransportSocket{
			Name: wellknown.TransportSocketTls,
			ConfigType: &envoy_api_v2_core.TransportSocket_TypedConfig{
				TypedConfig: marshalledDownstreamTLSContext,
			},
		},
	}

	return filterChain, nil
}
