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
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/envoy/rds/route"
	"github.com/openservicemesh/osm/pkg/errcode"
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

func (lb *listenerBuilder) newIngressHTTPFilterChain(cfg configurator.Configurator, svc service.MeshService, svcPort uint32) *xds_listener.FilterChain {
	marshalledDownstreamTLSContext, err := ptypes.MarshalAny(envoy.GetDownstreamTLSContext(lb.serviceIdentity, false /* TLS */))
	if err != nil {
		log.Error().Err(err).Str(errcode.Kind, errcode.ErrMarshallingXDSResource.String()).
			Msgf("Error marshalling DownstreamTLSContext object for proxy %s", svc)
		return nil
	}

	// Build the HTTP Connection Manager filter from its options
	ingressConnManager, err := httpConnManagerOptions{
		direction:         inbound,
		rdsRoutConfigName: route.IngressRouteConfigName,

		// Additional filters
		wasmStatsHeaders: nil, // no WASM Stats for ingress traffic
		extAuthConfig:    lb.getExtAuthConfig(),

		// Tracing options
		enableTracing:      lb.cfg.IsTracingEnabled(),
		tracingAPIEndpoint: lb.cfg.GetTracingEndpoint(),
	}.build()
	if err != nil {
		log.Error().Err(err).Msgf("Error building inbound HTTP connection manager for proxy with identity %s and service %s", lb.serviceIdentity, svc)
		return nil
	}

	marshalledIngressConnManager, err := ptypes.MarshalAny(ingressConnManager)
	if err != nil {
		log.Error().Err(err).Str(errcode.Kind, errcode.ErrMarshallingXDSResource.String()).
			Msgf("Error marshalling inbound HTTP connection manager for proxy with identity %s and service %s", lb.serviceIdentity, svc)
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
					TypedConfig: marshalledIngressConnManager,
				},
			},
		},
	}
}

func (lb *listenerBuilder) getIngressFilterChains(svc service.MeshService) []*xds_listener.FilterChain {
	var ingressFilterChains []*xds_listener.FilterChain

	protocolToPortMap, err := lb.meshCatalog.GetTargetPortToProtocolMappingForService(svc)
	if err != nil {
		log.Error().Err(err).Str(errcode.Kind, errcode.ErrGettingServicePorts.String()).
			Msgf("Error retrieving port to protocol mapping for service %s", svc)
		return ingressFilterChains
	}

	// Create protocol specific ingress filter chains per port to handle different ports serving different protocols
	for port, appProtocol := range protocolToPortMap {
		switch appProtocol {
		case constants.ProtocolHTTP:
			// Ingress filter chain for HTTP port
			if lb.cfg.UseHTTPSIngress() {
				// Filter chain with SNI matching enabled for HTTPS clients that set the SNI
				ingressFilterChainWithSNI := lb.newIngressHTTPFilterChain(lb.cfg, svc, port)
				ingressFilterChainWithSNI.Name = fmt.Sprintf("%s:%d", inboundIngressHTTPSFilterChain, port)
				ingressFilterChainWithSNI.FilterChainMatch.ServerNames = []string{svc.ServerName()}
				ingressFilterChains = append(ingressFilterChains, ingressFilterChainWithSNI)
			}

			// Filter chain without SNI matching enabled for HTTP clients and HTTPS clients that don't set the SNI
			ingressFilterChainWithoutSNI := lb.newIngressHTTPFilterChain(lb.cfg, svc, port)
			ingressFilterChainWithoutSNI.Name = fmt.Sprintf("%s:%d", inboundIngressNonSNIFilterChain, port)
			ingressFilterChains = append(ingressFilterChains, ingressFilterChainWithoutSNI)

		default:
			log.Error().Str(errcode.Kind, errcode.ErrUnsupportedProtocolForService.String()).
				Msgf("Cannot build ingress filter chain. Protocol %s is not supported for service %s on port %d",
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
