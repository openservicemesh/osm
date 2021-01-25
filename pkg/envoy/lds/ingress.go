package lds

import (
	"fmt"

	xds_core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	xds_listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/envoy/route"
	"github.com/openservicemesh/osm/pkg/service"
)

const (
	// inboundIngressHTTPSFilterChain is the name of the ingress filter chain to handle HTTPS traffic with SNI set
	inboundIngressHTTPSFilterChain = "inbound-ingress-https-filter-chain"

	// inboundIngressNonSNIFilterChain is the name of the ingress filter chain that handles either HTTP or HTTPS traffic without SNI set
	inboundIngressNonSNIFilterChain = "inbound-ingress-non-sni-filter-chain"
)

func getIngressTransportProtocol(forHTTPS bool) string {
	if forHTTPS {
		return envoy.TransportProtocolTLS
	}
	return ""
}

func newIngressHTTPFilterChain(cfg configurator.Configurator, svc service.MeshService, svcPort uint32) *xds_listener.FilterChain {
	marshalledDownstreamTLSContext, err := ptypes.MarshalAny(envoy.GetDownstreamTLSContext(svc, false /* TLS */))
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
			DestinationPort: &wrapperspb.UInt32Value{
				Value: svcPort,
			},
			TransportProtocol: getIngressTransportProtocol(cfg.UseHTTPSIngress()),
		},
		TransportSocket: getIngressTransportSocket(cfg.UseHTTPSIngress(), marshalledDownstreamTLSContext),
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

func (lb *listenerBuilder) getIngressFilterChains(svc service.MeshService) []*xds_listener.FilterChain {
	var ingressFilterChains []*xds_listener.FilterChain

	protocolToPortMap, err := lb.meshCatalog.GetTargetPortToProtocolMappingForService(svc)
	if err != nil {
		log.Error().Err(err).Msgf("Error retrieving port to protocol mapping for service %s", svc)
		return ingressFilterChains
	}

	// Create protocol specific inbound filter chains per port to handle different ports serving different protocols
	for port, appProtocol := range protocolToPortMap {
		switch appProtocol {
		case httpAppProtocol:
			// Ingress filter chain for HTTP port
			if lb.cfg.UseHTTPSIngress() {
				// Filter chain with SNI matching enabled for HTTPS clients that set the SNI
				ingressFilterChainWithSNI := newIngressHTTPFilterChain(lb.cfg, svc, port)
				ingressFilterChainWithSNI.Name = fmt.Sprintf("%s:%d", inboundIngressHTTPSFilterChain, port)
				ingressFilterChainWithSNI.FilterChainMatch.ServerNames = []string{svc.ServerName()}
				ingressFilterChains = append(ingressFilterChains, ingressFilterChainWithSNI)
			}

			// Filter chain without SNI matching enabled for HTTP clients and HTTPS clients that don't set the SNI
			ingressFilterChainWithoutSNI := newIngressHTTPFilterChain(lb.cfg, svc, port)
			ingressFilterChainWithoutSNI.Name = fmt.Sprintf("%s:%d", inboundIngressNonSNIFilterChain, port)
			ingressFilterChains = append(ingressFilterChains, ingressFilterChainWithoutSNI)

		default:
			log.Error().Msgf("Cannot build inbound filter chain. Protocol %s is not supported for service %s on port %d",
				appProtocol, svc, port)
		}
	}

	return ingressFilterChains
}

func getIngressTransportSocket(forHTTPS bool, marshalledDownstreamTLSContext *any.Any) *xds_core.TransportSocket {
	if forHTTPS {
		return &xds_core.TransportSocket{
			Name: wellknown.TransportSocketTls,
			ConfigType: &xds_core.TransportSocket_TypedConfig{
				TypedConfig: marshalledDownstreamTLSContext,
			},
		}
	}
	return nil
}
