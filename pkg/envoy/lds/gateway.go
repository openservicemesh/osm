package lds

import (
	"fmt"

	xds_listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	xds_tcp_proxy "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/tcp_proxy/v3"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/service"
)

const (
	multiclusterGatewayFilterChainName = "multicluster-gateway-filter-chain"

	// multiclusterGatewayListenerPort is port number for the multicluster gateway.
	multiclusterGatewayListenerPort = 15443
)

func (lb *listenerBuilder) buildMulticlusterGatewayListener() (*xds_listener.Listener, error) {
	upstreamServices := lb.meshCatalog.ListOutboundServicesForMulticlusterGateway()
	filterChains, err := getMulticlusterGatewayFilterChains(upstreamServices)
	if err != nil {
		log.Error().Err(err).Str(constants.LogFieldContext, constants.LogContextMulticluster).Msg("[Multicluster] Error creating Multicluster gateway filter chain")
		return nil, err
	}

	return &xds_listener.Listener{
		Name:         multiclusterListenerName,
		Address:      envoy.GetAddress(constants.WildcardIPAddr, multiclusterGatewayListenerPort),
		FilterChains: filterChains,
		ListenerFilters: []*xds_listener.ListenerFilter{
			{
				Name: wellknown.TlsInspector,
			},
		},
	}, nil
}

func getMulticlusterGatewayFilterChains(upstreamServices []service.MeshService) ([]*xds_listener.FilterChain, error) {
	var filterChains []*xds_listener.FilterChain
	for _, upstreamSvc := range upstreamServices {
		tcpProxy := &xds_tcp_proxy.TcpProxy{
			StatPrefix:       upstreamSvc.String(),
			ClusterSpecifier: &xds_tcp_proxy.TcpProxy_Cluster{Cluster: upstreamSvc.String()},
			AccessLog:        envoy.GetAccessLog(),
		}

		marshalledTCPProxy, err := anypb.New(tcpProxy)
		if err != nil {
			log.Error().Err(err).Msgf("[Multicluster] Error marshalling tcpProxy object for gateway filter chain service %s", upstreamSvc.String())
			continue
		}

		filterChain := &xds_listener.FilterChain{
			Name: fmt.Sprintf("%s-%s", multiclusterGatewayFilterChainName, upstreamSvc.Name),
			FilterChainMatch: &xds_listener.FilterChainMatch{
				ServerNames: []string{
					upstreamSvc.ServerName(),
				},
				TransportProtocol:    envoy.TransportProtocolTLS,
				ApplicationProtocols: envoy.ALPNInMesh, // in-mesh proxies will advertise this, set in UpstreamTlsContext
			},
			Filters: []*xds_listener.Filter{
				{
					Name: "envoy.filters.network.sni_cluster",
				},
				{
					Name: wellknown.TCPProxy,
					ConfigType: &xds_listener.Filter_TypedConfig{
						TypedConfig: marshalledTCPProxy,
					},
				},
			},
		}
		filterChains = append(filterChains, filterChain)
	}
	return filterChains, nil
}
