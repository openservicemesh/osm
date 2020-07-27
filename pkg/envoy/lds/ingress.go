package lds

import (
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/listener"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/open-service-mesh/osm/pkg/configurator"
	"github.com/open-service-mesh/osm/pkg/envoy"
	"github.com/open-service-mesh/osm/pkg/envoy/route"
	"github.com/open-service-mesh/osm/pkg/service"
)

func getIngressTransportProtocol(cfg configurator.Configurator) string {
	if cfg.UseHTTPSIngress() {
		return envoy.TransportProtocolTLS
	}
	return ""
}

func getIngressFilterChain(cfg configurator.Configurator, svc service.NamespacedService) *envoy_api_v2_listener.FilterChain {
	marshalledDownstreamTLSContext, err := envoy.MessageToAny(envoy.GetDownstreamTLSContext(svc, false /* TLS */))
	if err != nil {
		log.Error().Err(err).Msgf("Error marshalling DownstreamTLSContext object for proxy %s", svc)
		return nil
	}

	inboundConnManager := getHTTPConnectionManager(route.InboundRouteConfigName, cfg)
	marshalledInboundConnManager, err := ptypes.MarshalAny(inboundConnManager)
	if err != nil {
		log.Error().Err(err).Msgf("Error marshalling inbound HttpConnectionManager object for proxy %s", svc)
		return nil
	}

	return &envoy_api_v2_listener.FilterChain{
		// Filter chain with SNI matching enabled for clients that set the SNI
		FilterChainMatch: &envoy_api_v2_listener.FilterChainMatch{
			TransportProtocol: getIngressTransportProtocol(cfg),
		},
		TransportSocket: getIngressTransportSocket(cfg, marshalledDownstreamTLSContext),
		Filters: []*envoy_api_v2_listener.Filter{
			{
				Name: wellknown.HTTPConnectionManager,
				ConfigType: &envoy_api_v2_listener.Filter_TypedConfig{
					TypedConfig: marshalledInboundConnManager,
				},
			},
		},
	}
}

func getInboundIngressFilterChains(svc service.NamespacedService, cfg configurator.Configurator) []*envoy_api_v2_listener.FilterChain {
	// Filter chain without SNI matching enabled for clients that don't set the SNI
	ingressFilterChainWithoutSNI := getIngressFilterChain(cfg, svc)

	// Filter chain with SNI matching enabled for clients that set the SNI
	ingressFilterChainWithSNI := getIngressFilterChain(cfg, svc)
	ingressFilterChainWithSNI.FilterChainMatch.ServerNames = []string{svc.GetCommonName().String()}

	return []*envoy_api_v2_listener.FilterChain{
		ingressFilterChainWithSNI,
		ingressFilterChainWithoutSNI,
	}
}

func getIngressTransportSocket(cfg configurator.Configurator, marshalledDownstreamTLSContext *any.Any) *envoy_api_v2_core.TransportSocket {
	if cfg.UseHTTPSIngress() {
		return &envoy_api_v2_core.TransportSocket{
			Name: wellknown.TransportSocketTls,
			ConfigType: &envoy_api_v2_core.TransportSocket_TypedConfig{
				TypedConfig: marshalledDownstreamTLSContext,
			},
		}
	}
	return nil
}
