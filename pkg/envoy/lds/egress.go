package lds

import (
	"fmt"

	xds_core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	xds_listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	xds_tcp_proxy "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/tcp_proxy/v3"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/envoy/rds/route"
	"github.com/openservicemesh/osm/pkg/errcode"
	"github.com/openservicemesh/osm/pkg/trafficpolicy"
)

const (
	egressHTTPFilterChainPrefix = "egress-http"
	egressTCPFilterChainPrefix  = "egress-tcp"
)

var (
	// httpProtocols is the list of allowed HTTP protocols that the downstream can use in
	// an HTTP request that will be subjected to HTTP routing rules.
	// *Note: HTTP2 over TLS (h2) traffic is not subjected to HTTP based routing rules because the
	// traffic is encrypted and will be proxied as a TCP stream instead.
	httpProtocols = []string{"http/1.0", "http/1.1", "h2c"}
)

// getEgressFilterChainsForMatches returns a slice of egress filter chains for the given traffic matches
func (lb *listenerBuilder) getEgressFilterChainsForMatches(matches []*trafficpolicy.TrafficMatch) []*xds_listener.FilterChain {
	var filterChains []*xds_listener.FilterChain

	for _, match := range matches {
		switch match.DestinationProtocol {
		case constants.ProtocolHTTP:
			// HTTP protocol --> HTTPConnectionManager filter
			if filterChain, err := lb.getEgressHTTPFilterChain(match.DestinationPort); err != nil {
				log.Error().Err(err).Msgf("Error building egress HTTP filter chain for port [%d]", match.DestinationPort)
			} else {
				filterChains = append(filterChains, filterChain)
			}

		case constants.ProtocolTCP, constants.ProtocolHTTPS, constants.ProtocolTCPServerFirst:
			// TCP or HTTPS protocol --> TCPProxy filter
			if filterChain, err := lb.getEgressTCPFilterChain(*match); err != nil {
				log.Error().Err(err).Msgf("Error building egress filter chain for match [%v]", *match)
			} else {
				filterChains = append(filterChains, filterChain)
			}
		}
	}

	return filterChains
}

func (lb *listenerBuilder) getEgressHTTPFilterChain(destinationPort int) (*xds_listener.FilterChain, error) {
	filter, err := lb.getOutboundHTTPFilter(route.GetEgressRouteConfigNameForPort(destinationPort))
	if err != nil {
		log.Error().Err(err).Msgf("Error building HTTP filter chain for destination port [%d]", destinationPort)
		return nil, err
	}

	filterChainName := fmt.Sprintf("%s.%d", egressHTTPFilterChainPrefix, destinationPort)
	return &xds_listener.FilterChain{
		Name:    filterChainName,
		Filters: []*xds_listener.Filter{filter},
		FilterChainMatch: &xds_listener.FilterChainMatch{
			DestinationPort: &wrapperspb.UInt32Value{
				Value: uint32(destinationPort),
			},
			ApplicationProtocols: httpProtocols,
		},
	}, nil
}

func (lb *listenerBuilder) getEgressTCPFilterChain(match trafficpolicy.TrafficMatch) (*xds_listener.FilterChain, error) {
	tcpProxy := &xds_tcp_proxy.TcpProxy{
		StatPrefix:       fmt.Sprintf("%s.%d", egressTCPProxyStatPrefix, match.DestinationPort),
		ClusterSpecifier: &xds_tcp_proxy.TcpProxy_Cluster{Cluster: match.Cluster},
	}

	marshalledTCPProxy, err := anypb.New(tcpProxy)
	if err != nil {
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrMarshallingXDSResource)).
			Msgf("Error marshalling TcpProxy for TrafficMatch %v", match)
		return nil, err
	}

	tcpFilter := &xds_listener.Filter{
		Name:       wellknown.TCPProxy,
		ConfigType: &xds_listener.Filter_TypedConfig{TypedConfig: marshalledTCPProxy},
	}

	var destinationPrefixes []*xds_core.CidrRange
	for _, ipRange := range match.DestinationIPRanges {
		cidr, err := envoy.GetCIDRRangeFromStr(ipRange)
		if err != nil {
			log.Error().Err(err).Str(errcode.Kind, errcode.ErrInvalidEgressIPRange.String()).
				Msgf("Error parsing IP range %s while building TCP filter chain for match %v, skipping", ipRange, match)
			continue
		}
		destinationPrefixes = append(destinationPrefixes, cidr)
	}

	return &xds_listener.FilterChain{
		Name:    fmt.Sprintf("%s.%d", egressTCPFilterChainPrefix, match.DestinationPort),
		Filters: []*xds_listener.Filter{tcpFilter},
		FilterChainMatch: &xds_listener.FilterChainMatch{
			DestinationPort: &wrapperspb.UInt32Value{
				Value: uint32(match.DestinationPort),
			},
			ServerNames:  match.ServerNames,
			PrefixRanges: destinationPrefixes,
		},
	}, nil
}
