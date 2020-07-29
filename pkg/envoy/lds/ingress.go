package lds

import (
	envoy_api_v2_core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	envoy_api_v2_listener "github.com/envoyproxy/go-control-plane/envoy/api/v2/listener"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/envoy/route"
	"github.com/openservicemesh/osm/pkg/service"
)

func getIngressTransportProtocol(cfg configurator.Configurator) string {
	if cfg.UseHTTPSIngress() {
		return envoy.TransportProtocolTLS
	}
	return ""
}

func newIngressFilterChain(cfg configurator.Configurator, svc service.NamespacedService) *envoy_api_v2_listener.FilterChain {
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

func getIngressFilterChains(svc service.NamespacedService, cfg configurator.Configurator) []*envoy_api_v2_listener.FilterChain {
	var ingressFilterChains []*envoy_api_v2_listener.FilterChain

	if cfg.UseHTTPSIngress() {
		// Filter chain with SNI matching enabled for HTTPS clients that set the SNI
		ingressFilterChainWithSNI := newIngressFilterChain(cfg, svc)
		ingressFilterChainWithSNI.FilterChainMatch.ServerNames = []string{svc.GetCommonName().String()}
		ingressFilterChains = append(ingressFilterChains, ingressFilterChainWithSNI)
	}

	// Filter chain without SNI matching enabled for HTTP clients and HTTPS clients that don't set the SNI
	ingressFilterChainWithoutSNI := newIngressFilterChain(cfg, svc)
	ingressFilterChains = append(ingressFilterChains, ingressFilterChainWithoutSNI)

	return ingressFilterChains
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
