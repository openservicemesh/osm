package lds

import (
	xds "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	envoy_api_v2_core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	listener "github.com/envoyproxy/go-control-plane/envoy/api/v2/listener"
	envoy_hcm "github.com/envoyproxy/go-control-plane/envoy/config/filter/network/http_connection_manager/v2"
	tcp_proxy "github.com/envoyproxy/go-control-plane/envoy/config/filter/network/tcp_proxy/v2"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"github.com/golang/protobuf/ptypes"

	"github.com/open-service-mesh/osm/pkg/constants"
	"github.com/open-service-mesh/osm/pkg/envoy"
)

func buildOutboundListener(connManager *envoy_hcm.HttpConnectionManager, withEgress bool) (*xds.Listener, error) {
	marshalledConnManager, err := ptypes.MarshalAny(connManager)
	if err != nil {
		log.Error().Err(err).Msgf("Error marshalling HttpConnectionManager object")
		return nil, err
	}

	listener := &xds.Listener{
		Name:             outboundListenerName,
		Address:          envoy.GetAddress(constants.WildcardIPAddr, constants.EnvoyOutboundListenerPort),
		TrafficDirection: envoy_api_v2_core.TrafficDirection_OUTBOUND,
		FilterChains: []*listener.FilterChain{
			{
				Filters: []*listener.Filter{
					{
						Name: wellknown.HTTPConnectionManager,
						ConfigType: &listener.Filter_TypedConfig{
							TypedConfig: marshalledConnManager,
						},
					},
				},
			},
		},
	}

	if withEgress {
		// With egress, a filter chain to match TLS traffic is added to the outbound listener.
		// In-mesh traffic will always be HTTP so this filter chain will not match for in-mesh.
		// HTTPS egress traffic will match this filter chain and will be proxied to its original
		// destination.
		egressFilterChain, err := buildEgressHTTPSFilterChain()
		if err != nil {
			return nil, err
		}
		listener.FilterChains = append(listener.FilterChains, egressFilterChain)
		listenerFilters := buildEgressListenerFilters()
		listener.ListenerFilters = append(listener.ListenerFilters, listenerFilters...)
	}

	return listener, nil
}

func buildInboundListener() *xds.Listener {
	return &xds.Listener{
		Name:             inboundListenerName,
		Address:          envoy.GetAddress(constants.WildcardIPAddr, constants.EnvoyInboundListenerPort),
		TrafficDirection: envoy_api_v2_core.TrafficDirection_INBOUND,
		FilterChains:     []*listener.FilterChain{},
		ListenerFilters: []*listener.ListenerFilter{
			{
				Name: wellknown.TlsInspector,
			},
		},
	}
}

func buildPrometheusListener(connManager *envoy_hcm.HttpConnectionManager) (*xds.Listener, error) {
	marshalledConnManager, err := ptypes.MarshalAny(connManager)
	if err != nil {
		log.Error().Err(err).Msgf("Error marshalling HttpConnectionManager object")
		return nil, err
	}

	listener := &xds.Listener{
		Name:             prometheusListenerName,
		TrafficDirection: envoy_api_v2_core.TrafficDirection_INBOUND,
		Address:          envoy.GetAddress(constants.WildcardIPAddr, constants.EnvoyPrometheusInboundListenerPort),
		FilterChains: []*listener.FilterChain{
			{
				Filters: []*listener.Filter{
					{
						Name: wellknown.HTTPConnectionManager,
						ConfigType: &listener.Filter_TypedConfig{
							TypedConfig: marshalledConnManager,
						},
					},
				},
			},
		},
	}
	return listener, nil
}

func buildEgressHTTPSFilterChain() (*listener.FilterChain, error) {
	tcpProxy := &tcp_proxy.TcpProxy{
		StatPrefix:       envoy.OutboundPassthroughCluster,
		ClusterSpecifier: &tcp_proxy.TcpProxy_Cluster{Cluster: envoy.OutboundPassthroughCluster},
	}
	marshalledTCPProxy, err := envoy.MessageToAny(tcpProxy)
	if err != nil {
		log.Error().Err(err).Msgf("Error marshalling TcpProxy object for egress HTTPS filter chain")
		return nil, err
	}

	return &listener.FilterChain{
		FilterChainMatch: &listener.FilterChainMatch{
			TransportProtocol: envoy.TransportProtocolTLS,
		},
		Filters: []*listener.Filter{
			{
				Name:       wellknown.TCPProxy,
				ConfigType: &listener.Filter_TypedConfig{TypedConfig: marshalledTCPProxy},
			},
		},
	}, nil
}

func buildEgressListenerFilters() []*listener.ListenerFilter {
	return []*listener.ListenerFilter{
		{
			// The OriginalDestination ListenerFilter is used to redirect traffic
			// to its original destination.
			Name: wellknown.OriginalDestination,
		},
		{
			// The TlsInspector ListenerFilter is used to examine the transport protocol
			Name: wellknown.TlsInspector,
		},
	}
}
