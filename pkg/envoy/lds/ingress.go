package lds

import (
	"fmt"
	"strings"

	xds_core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	xds_listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/envoy/rds/route"
	"github.com/openservicemesh/osm/pkg/errcode"
	"github.com/openservicemesh/osm/pkg/trafficpolicy"
)

func (lb *listenerBuilder) buildIngressFilterChains() []*xds_listener.FilterChain {
	if lb.ingressTrafficPolicies == nil {
		return nil
	}

	var filterChains []*xds_listener.FilterChain

	for _, policy := range lb.ingressTrafficPolicies {
		for _, trafficMatch := range policy.TrafficMatches {
			filterChain, err := lb.buildIngressFilterChainFromTrafficMatch(trafficMatch)
			if err != nil {
				log.Error().Err(err).Msgf("Error building ingress filter chain for traffic match %s for proxy with identity %s", trafficMatch.Name, lb.proxyIdentity)
				continue
			}
			filterChains = append(filterChains, filterChain)
		}
	}

	return filterChains
}

func (lb *listenerBuilder) buildIngressFilterChainFromTrafficMatch(trafficMatch *trafficpolicy.IngressTrafficMatch) (*xds_listener.FilterChain, error) {
	if trafficMatch == nil {
		return nil, fmt.Errorf("Nil IngressTrafficMatch for ingress on proxy with identity %s", lb.proxyIdentity)
	}

	hcmBuilder := HTTPConnManagerBuilder()
	hcmBuilder.StatsPrefix(route.IngressRouteConfigName).
		RouteConfigName(route.IngressRouteConfigName)

	if lb.httpTracingEndpoint != "" {
		tracing, err := getHTTPTracingConfig(lb.httpTracingEndpoint)
		if err != nil {
			return nil, fmt.Errorf("error building outbound http filter: %w", err)
		}
		hcmBuilder.Tracing(tracing)
	}
	if lb.extAuthzConfig != nil && lb.extAuthzConfig.Enable {
		hcmBuilder.AddFilter(getExtAuthzHTTPFilter(lb.extAuthzConfig))
	}

	// Build the HTTP Connection Manager filter
	hcmFilter, err := hcmBuilder.Build()
	if err != nil {
		return nil, fmt.Errorf("error building ingress filter chain: %w", err)
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
			hcmFilter,
		},
	}

	switch strings.ToLower(trafficMatch.Protocol) {
	case constants.ProtocolHTTP:
		// For HTTP backend, only allow traffic from authorized
		if filterChain.FilterChainMatch.SourcePrefixRanges == nil {
			log.Warn().Msgf("Allowing HTTP ingress on proxy with identity %s is insecure, use IngressBackend.Spec.Sources to restrict clients", lb.proxyIdentity)
		}

	case constants.ProtocolHTTPS:
		// For HTTPS backend, configure the following:
		// 1. TransportProtocol to match TLS
		// 2. ServerNames (SNI)
		// 3. TransportSocket to terminate TLS from downstream connections
		filterChain.FilterChainMatch.TransportProtocol = envoy.TransportProtocolTLS
		filterChain.FilterChainMatch.ServerNames = trafficMatch.ServerNames

		marshalledDownstreamTLSContext, err := anypb.New(envoy.GetDownstreamTLSContext(lb.proxyIdentity, !trafficMatch.SkipClientCertValidation, lb.sidecarSpec))
		if err != nil {
			return nil, fmt.Errorf("Error marshalling DownstreamTLSContext in ingress filter chain for proxy with identity %s", lb.proxyIdentity)
		}

		filterChain.TransportSocket = &xds_core.TransportSocket{
			Name: trafficMatch.Name,
			ConfigType: &xds_core.TransportSocket_TypedConfig{
				TypedConfig: marshalledDownstreamTLSContext,
			},
		}

	default:
		err := fmt.Errorf("Unsupported ingress protocol %s on proxy with identity %s. Ingress protocol must be one of 'http, https'", trafficMatch.Protocol, lb.proxyIdentity)
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrUnsupportedProtocolForService)).Msg("Error building filter chain for ingress")
		return nil, err
	}

	return filterChain, nil
}
