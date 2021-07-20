package lds

import (
	xds_core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	xds_listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	xds_tcp_proxy "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/tcp_proxy/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"github.com/golang/protobuf/ptypes"

	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/identity"
)

const (
	inboundMulticlusterGatewayFilterChainName = "inbound-multicluster-gateway-filter-chain"
)

func (lb *listenerBuilder) buildGatewayListeners() []types.Resource {
	if !lb.cfg.GetFeatureFlags().EnableMulticlusterMode {
		return nil
	}

	// TODO(draychev): What should the Service Identity here be?
	filterChain, err := getGatewayFilterChain("osm-gateway.osm-system")
	if err != nil {
		log.Err(err).Msg("[Multicluster] Error creating Multicluster gateway filter chain")
	}

	return []types.Resource{
		&xds_listener.Listener{
			Name:         multiclusterListenerName,
			Address:      envoy.GetAddress(constants.WildcardIPAddr, constants.EnvoyInboundListenerPort),
			FilterChains: []*xds_listener.FilterChain{filterChain},
		},
	}
}

func getGatewayFilterChain(svcIdent identity.ServiceIdentity) (*xds_listener.FilterChain, error) {
	tcpProxy := &xds_tcp_proxy.TcpProxy{
		StatPrefix:       envoy.MulticlusterGatewayCluster,
		ClusterSpecifier: &xds_tcp_proxy.TcpProxy_Cluster{Cluster: envoy.MulticlusterGatewayCluster},
		AccessLog:        envoy.GetAccessLog(),
	}
	marshalledTCPProxy, err := ptypes.MarshalAny(tcpProxy)
	if err != nil {
		log.Error().Err(err).Msg("[Multicluster] Error marshalling tcpProxy object for gateway filter chain")
		return nil, err
	}

	marshalledDownstreamTLSContext, err := ptypes.MarshalAny(envoy.GetDownstreamTLSContext(svcIdent, true /* mTLS */))
	if err != nil {
		log.Error().Err(err).Msgf("[Multicluster] Error marshalling DownstreamTLSContext object for service identity %s", svcIdent)
		return nil, err
	}

	filterChain := &xds_listener.FilterChain{
		Name: inboundMulticlusterGatewayFilterChainName,
		FilterChainMatch: &xds_listener.FilterChainMatch{
			ServerNames: []string{
				"*.local",
			},
			TransportProtocol:    envoy.TransportProtocolTLS,
			ApplicationProtocols: envoy.ALPNInMesh, // in-mesh proxies will advertise this, set in UpstreamTlsContext
		},
		Filters: []*xds_listener.Filter{
			{
				// This is of utmost importance to the Multicluster Gateway!
				Name: "envoy.filters.network.sni_cluster",
			},
			{
				Name: wellknown.TCPProxy,
				ConfigType: &xds_listener.Filter_TypedConfig{
					TypedConfig: marshalledTCPProxy,
				},
			},
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
