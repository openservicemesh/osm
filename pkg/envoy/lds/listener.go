package lds

import (
	xds "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	auth "github.com/envoyproxy/go-control-plane/envoy/api/v2/auth"
	envoy_api_v2_core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	listener "github.com/envoyproxy/go-control-plane/envoy/api/v2/listener"
	envoy_hcm "github.com/envoyproxy/go-control-plane/envoy/config/filter/network/http_connection_manager/v2"
	tcp_proxy "github.com/envoyproxy/go-control-plane/envoy/config/filter/network/tcp_proxy/v2"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/wrappers"

	"github.com/open-service-mesh/osm/pkg/configurator"
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

func buildPrometheusListener(cfg configurator.Configurator) (*xds.Listener, error) {

	promConnMngr := getPrometheusConnectionManager()
	marshalledConnManager, err := ptypes.MarshalAny(promConnMngr)
	if err != nil {
		log.Error().Err(err).Msgf("Error marshalling HttpConnectionManager object")
		return nil, err
	}

	// Create the Listener
	prometheusListener := &xds.Listener{
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
		ListenerFilters: []*listener.ListenerFilter{
			{
				Name: wellknown.TlsInspector,
			},
		},
	}

	// Create the TLS context
	downstreamTLSContext := &auth.DownstreamTlsContext{
		CommonTlsContext: &auth.CommonTlsContext{
			TlsParams: envoy.GetTLSParams(),
			TlsCertificateSdsSecretConfigs: []*auth.SdsSecretConfig{{
				Name: envoy.SDSCert{
					Service:  envoy.GetPrometheusNamespacedService(cfg),
					CertType: envoy.ServiceCertType,
				}.String(),
				SdsConfig: envoy.GetADSConfigSource(),
			}},
		},

		// We do not authenticate client at this time
		RequireClientCertificate: &wrappers.BoolValue{Value: false},
	}
	marshalledDownstreamTLSContext, err := envoy.MessageToAny(downstreamTLSContext)
	if err != nil {
		log.Error().Err(err).Msgf("Error marshalling DownstreamTLSContext object for proxy %s", constants.EnvoyMetricsCluster)
		return nil, err
	}

	// Create the filter chain, attach TLS context
	prometheusFilterChain := &listener.FilterChain{
		FilterChainMatch: &listener.FilterChainMatch{
			// TODO: `ServerNames ?`
			TransportProtocol: envoy.TransportProtocolTLS,
		},
		TransportSocket: &envoy_api_v2_core.TransportSocket{
			Name: envoy.TransportSocketTLS,
			ConfigType: &envoy_api_v2_core.TransportSocket_TypedConfig{
				TypedConfig: marshalledDownstreamTLSContext,
			},
		},
		Filters: []*listener.Filter{
			{
				Name: wellknown.HTTPConnectionManager,
				ConfigType: &listener.Filter_TypedConfig{
					TypedConfig: marshalledConnManager,
				},
			},
		},
	}

	prometheusListener.FilterChains = append(prometheusListener.FilterChains, prometheusFilterChain)
	return prometheusListener, nil
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
