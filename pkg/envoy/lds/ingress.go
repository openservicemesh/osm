package lds

import (
	"strings"

	xds_core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	xds_listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	configv1alpha3 "github.com/openservicemesh/osm/pkg/apis/config/v1alpha3"

	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/envoy/rds/route"
	"github.com/openservicemesh/osm/pkg/errcode"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/trafficpolicy"
)

func (lb *listenerBuilder) getIngressFilterChains(svc service.MeshService) []*xds_listener.FilterChain {
	ingressPolicy, err := lb.meshCatalog.GetIngressTrafficPolicy(svc)
	if err != nil {
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrIngressFilterChain)).
			Msgf("Error getting ingress filter chain for proxy with identity %s and service %s", lb.serviceIdentity, svc)
		return nil
	}

	if ingressPolicy == nil {
		log.Trace().Msgf("No ingress policy confiugred for proxy with identity %s and service %s", lb.serviceIdentity, svc)
		return nil
	}

	var filterChains []*xds_listener.FilterChain
	for _, trafficMatch := range ingressPolicy.TrafficMatches {
		if filterChain, err := lb.getIngressFilterChainFromTrafficMatch(trafficMatch, lb.cfg.GetMeshConfig().Spec.Sidecar); err != nil {
			log.Error().Err(err).Msgf("Error building ingress filter chain for proxy with identity %s service %s", lb.serviceIdentity, svc)
		} else {
			filterChains = append(filterChains, filterChain)
		}
	}

	return filterChains
}

func (lb *listenerBuilder) getIngressFilterChainFromTrafficMatch(trafficMatch *trafficpolicy.IngressTrafficMatch, sidecarSpec configv1alpha3.SidecarSpec) (*xds_listener.FilterChain, error) {
	if trafficMatch == nil {
		return nil, errors.Errorf("Nil IngressTrafficMatch for ingress on proxy with identity %s", lb.serviceIdentity)
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
		return nil, errors.Errorf("Error building inbound HTTP connection manager for proxy with identity %s, traffic match: %v ", lb.serviceIdentity, trafficMatch)
	}

	marshalledIngressConnManager, err := anypb.New(ingressConnManager)
	if err != nil {
		return nil, errors.Errorf("Error marshalling ingress HttpConnectionManager object for proxy with identity %s", lb.serviceIdentity)
	}

	var sourcePrefixes []*xds_core.CidrRange
	for _, ipRange := range trafficMatch.SourceIPRanges {
		cidr, err := envoy.GetCIDRRangeFromStr(ipRange)
		if err != nil {
			log.Error().Err(err).Msgf("Error parsing IP range %s while building Ingress filter chain for match %v, skipping", ipRange, trafficMatch)
			continue
		}
		sourcePrefixes = append(sourcePrefixes, cidr)
	}

	filterChain := &xds_listener.FilterChain{
		Name: trafficMatch.Name,
		FilterChainMatch: &xds_listener.FilterChainMatch{
			DestinationPort: &wrapperspb.UInt32Value{
				Value: uint32(trafficMatch.Port),
			},
			SourcePrefixRanges: sourcePrefixes,
		},
		Filters: []*xds_listener.Filter{
			{
				Name: wellknown.HTTPConnectionManager,
				ConfigType: &xds_listener.Filter_TypedConfig{
					TypedConfig: marshalledIngressConnManager,
				},
			},
		},
	}

	switch strings.ToLower(trafficMatch.Protocol) {
	case constants.ProtocolHTTP:
		// For HTTP backend, only allow traffic from authorized
		if filterChain.FilterChainMatch.SourcePrefixRanges == nil {
			log.Warn().Msgf("Allowing HTTP ingress on proxy with identity %s is insecure, use IngressBackend.Spec.Sources to restrict clients", lb.serviceIdentity)
		}

	case constants.ProtocolHTTPS:
		// For HTTPS backend, configure the following:
		// 1. TransportProtocol to match TLS
		// 2. ServerNames (SNI)
		// 3. TransportSocket to terminate TLS from downstream connections
		filterChain.FilterChainMatch.TransportProtocol = envoy.TransportProtocolTLS
		filterChain.FilterChainMatch.ServerNames = trafficMatch.ServerNames

		marshalledDownstreamTLSContext, err := anypb.New(envoy.GetDownstreamTLSContext(lb.serviceIdentity, !trafficMatch.SkipClientCertValidation, sidecarSpec))
		if err != nil {
			return nil, errors.Errorf("Error marshalling DownstreamTLSContext in ingress filter chain for proxy with identity %s", lb.serviceIdentity)
		}

		filterChain.TransportSocket = &xds_core.TransportSocket{
			Name: wellknown.TransportSocketTls,
			ConfigType: &xds_core.TransportSocket_TypedConfig{
				TypedConfig: marshalledDownstreamTLSContext,
			},
		}

	default:
		err := errors.Errorf("Unsupported ingress protocol %s on proxy with identity %s. Ingress protocol must be one of 'http, https'", trafficMatch.Protocol, lb.serviceIdentity)
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrUnsupportedProtocolForService)).Msg("Error building filter chain for ingress")
		return nil, err
	}

	return filterChain, nil
}
