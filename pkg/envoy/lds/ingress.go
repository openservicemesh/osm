package lds

import (
	xds_core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	xds_listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"

	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"

	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/envoy/route"
	"github.com/openservicemesh/osm/pkg/service"
)

const (
	inboundIngressHTTPSFilterChain = "inbound-ingress-https-filter-chain"
	inboundIngressHTTPFilterChain  = "inbound-ingress-http-filter-chain"
)

func getIngressTransportProtocol(cfg configurator.Configurator) string {
	if cfg.UseHTTPSIngress() {
		return envoy.TransportProtocolTLS
	}
	return ""
}

func newIngressFilterChain(cfg configurator.Configurator, svc service.MeshService) *xds_listener.FilterChain {
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

	return &xds_listener.FilterChain{
		// Filter chain with SNI matching enabled for clients that set the SNI
		FilterChainMatch: &xds_listener.FilterChainMatch{
			TransportProtocol: getIngressTransportProtocol(cfg),
		},
		TransportSocket: getIngressTransportSocket(cfg, marshalledDownstreamTLSContext),
		Filters: []*xds_listener.Filter{
			{
				Name: wellknown.HTTPConnectionManager,
				ConfigType: &xds_listener.Filter_TypedConfig{
					TypedConfig: marshalledInboundConnManager,
				},
			},
		},
	}
}

func getIngressFilterChains(svc service.MeshService, cfg configurator.Configurator) []*xds_listener.FilterChain {
	var ingressFilterChains []*xds_listener.FilterChain

	if cfg.UseHTTPSIngress() {
		// Filter chain with SNI matching enabled for HTTPS clients that set the SNI
		ingressFilterChainWithSNI := newIngressFilterChain(cfg, svc)
		ingressFilterChainWithSNI.Name = inboundIngressHTTPSFilterChain
		ingressFilterChainWithSNI.FilterChainMatch.ServerNames = []string{svc.ServerName()}
		ingressFilterChains = append(ingressFilterChains, ingressFilterChainWithSNI)
	}

	// Filter chain without SNI matching enabled for HTTP clients and HTTPS clients that don't set the SNI
	ingressFilterChainWithoutSNI := newIngressFilterChain(cfg, svc)
	ingressFilterChainWithoutSNI.Name = inboundIngressHTTPFilterChain
	ingressFilterChains = append(ingressFilterChains, ingressFilterChainWithoutSNI)

	return ingressFilterChains
}

func getIngressTransportSocket(cfg configurator.Configurator, marshalledDownstreamTLSContext *any.Any) *xds_core.TransportSocket {
	if cfg.UseHTTPSIngress() {
		return &xds_core.TransportSocket{
			Name: wellknown.TransportSocketTls,
			ConfigType: &xds_core.TransportSocket_TypedConfig{
				TypedConfig: marshalledDownstreamTLSContext,
			},
		}
	}
	return nil
}
