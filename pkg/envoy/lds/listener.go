package lds

import (
	"fmt"

	mapset "github.com/deckarep/golang-set"
	xds_core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	xds_listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	xds_hcm "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	xds_tcp_proxy "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/tcp_proxy/v3"
	xds_type "github.com/envoyproxy/go-control-plane/envoy/type/v3"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/errcode"
	"github.com/openservicemesh/osm/pkg/protobuf"
	"github.com/openservicemesh/osm/pkg/trafficpolicy"
)

const (
	// InboundListenerName is the name of the listener used for inbound traffic
	InboundListenerName = "inbound-listener"

	// OutboundListenerName is the name of the listener used for outbound traffic
	OutboundListenerName = "outbound-listener"

	prometheusListenerName        = "inbound-prometheus-listener"
	outboundEgressFilterChainName = "outbound-egress-filter-chain"
	egressTCPProxyStatPrefix      = "egress-tcp-proxy"
)

func buildPrometheusListener(connManager *xds_hcm.HttpConnectionManager) (*xds_listener.Listener, error) {
	marshalledConnManager, err := anypb.New(connManager)
	if err != nil {
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrMarshallingXDSResource)).
			Msgf("Error marshalling HttpConnectionManager object")
		return nil, err
	}

	return &xds_listener.Listener{
		Name:             prometheusListenerName,
		TrafficDirection: xds_core.TrafficDirection_INBOUND,
		Address:          envoy.GetAddress(constants.WildcardIPAddr, constants.EnvoyPrometheusInboundListenerPort),
		FilterChains: []*xds_listener.FilterChain{
			{
				Filters: []*xds_listener.Filter{
					{
						Name: envoy.HTTPConnectionManagerFilterName,
						ConfigType: &xds_listener.Filter_TypedConfig{
							TypedConfig: marshalledConnManager,
						},
					},
				},
			},
		},
	}, nil
}

// getDefaultPassthroughFilterChain returns a filter chain that matches any traffic, allowing such
// traffic to be proxied to its original destination via the OutboundPassthroughCluster.
func getDefaultPassthroughFilterChain() *xds_listener.FilterChain {
	tcpProxy := &xds_tcp_proxy.TcpProxy{
		StatPrefix:       fmt.Sprintf("%s.%s", egressTCPProxyStatPrefix, envoy.OutboundPassthroughCluster),
		ClusterSpecifier: &xds_tcp_proxy.TcpProxy_Cluster{Cluster: envoy.OutboundPassthroughCluster},
	}

	return &xds_listener.FilterChain{
		Name: outboundEgressFilterChainName,
		Filters: []*xds_listener.Filter{
			{
				Name:       envoy.TCPProxyFilterName,
				ConfigType: &xds_listener.Filter_TypedConfig{TypedConfig: protobuf.MustMarshalAny(tcpProxy)},
			},
		},
	}
}

// getFilterMatchPredicateForTrafficMatches returns a ListenerFilterChainMatchPredicate corresponding to server-first ports.
// If there are no server-first ports, a nil object is returned.
func getFilterMatchPredicateForTrafficMatches(matches []*trafficpolicy.TrafficMatch) *xds_listener.ListenerFilterChainMatchPredicate {
	var ports []int
	portSet := mapset.NewSet()

	for _, match := range matches {
		// Only configure match predicate for server first protocol
		if match.DestinationProtocol != constants.ProtocolTCPServerFirst {
			continue
		}

		newlyAdded := portSet.Add(match.DestinationPort)
		if newlyAdded {
			ports = append(ports, match.DestinationPort)
		}
	}

	if len(ports) == 0 {
		return nil
	}

	return getFilterMatchPredicateForPorts(ports)
}

// getFilterMatchPredicateForPorts returns a ListenerFilterChainMatchPredicate that matches the given set of ports
func getFilterMatchPredicateForPorts(ports []int) *xds_listener.ListenerFilterChainMatchPredicate {
	if len(ports) == 0 {
		return nil
	}

	var matchPredicates []*xds_listener.ListenerFilterChainMatchPredicate

	// Create a match predicate for each port
	for _, port := range ports {
		matchRule := &xds_listener.ListenerFilterChainMatchPredicate{
			Rule: &xds_listener.ListenerFilterChainMatchPredicate_DestinationPortRange{
				DestinationPortRange: &xds_type.Int32Range{
					Start: int32(port),     // Start is inclusive
					End:   int32(port + 1), // End is exclusive
				},
			},
		}
		matchPredicates = append(matchPredicates, matchRule)
	}

	if len(matchPredicates) > 1 {
		// Proto constraint validation requirers at least 2 items to be able
		// to use an OR based match set.
		return &xds_listener.ListenerFilterChainMatchPredicate{
			Rule: &xds_listener.ListenerFilterChainMatchPredicate_OrMatch{
				OrMatch: &xds_listener.ListenerFilterChainMatchPredicate_MatchSet{
					Rules: matchPredicates,
				},
			},
		}
	}

	return &xds_listener.ListenerFilterChainMatchPredicate{Rule: matchPredicates[0].GetRule()}
}
