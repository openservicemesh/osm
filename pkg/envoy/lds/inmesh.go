package lds

import (
	xds_core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	xds_listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"

	"github.com/golang/protobuf/ptypes"

	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/envoy/route"
	"github.com/openservicemesh/osm/pkg/service"
)

const (
	inboundMeshFilterChainName = "inbound-mesh-filter-chain"
)

func getInboundInMeshFilterChain(proxyServiceName service.MeshService, cfg configurator.Configurator) (*xds_listener.FilterChain, error) {
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

	filterChain := &xds_listener.FilterChain{
		Name: inboundMeshFilterChainName,
		Filters: []*xds_listener.Filter{
			{
				Name: wellknown.HTTPConnectionManager,
				ConfigType: &xds_listener.Filter_TypedConfig{
					TypedConfig: marshalledInboundConnManager,
				},
			},
		},

		// The 'FilterChainMatch' field defines the criteria for matching traffic against filters in this filter chain
		FilterChainMatch: &xds_listener.FilterChainMatch{
			// The ServerName is the SNI set by the downstream in the UptreamTlsContext by GetUpstreamTLSContext()
			// This is not a field obtained from the mTLS Certificate.
			ServerNames: []string{proxyServiceName.ServerName()},

			// Only match when transport protocol is TLS
			TransportProtocol: envoy.TransportProtocolTLS,

			// In-mesh proxies will advertise this, set in the UpstreamTlsContext by GetUpstreamTLSContext()
			ApplicationProtocols: envoy.ALPNInMesh,
		},

		TransportSocket: &xds_core.TransportSocket{
			Name: wellknown.TransportSocketTls,
			ConfigType: &xds_core.TransportSocket_TypedConfig{
				TypedConfig: marshalledDownstreamTLSContext,
			},
		},
	}

	return filterChain, nil
}
