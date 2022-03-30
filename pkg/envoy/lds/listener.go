package lds

import (
	"fmt"

	mapset "github.com/deckarep/golang-set"
	xds_core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	xds_listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	xds_hcm "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	xds_tcp_proxy "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/tcp_proxy/v3"
	xds_type "github.com/envoyproxy/go-control-plane/envoy/type/v3"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/errcode"
	"github.com/openservicemesh/osm/pkg/trafficpolicy"
)

const (
	inboundListenerName           = "inbound-listener"
	outboundListenerName          = "outbound-listener"
	multiclusterListenerName      = "multicluster-listener"
	prometheusListenerName        = "inbound-prometheus-listener"
	outboundEgressFilterChainName = "outbound-egress-filter-chain"
	egressTCPProxyStatPrefix      = "egress-tcp-proxy"
)

func (lb *listenerBuilder) newOutboundListener() (*xds_listener.Listener, error) {
	serviceFilterChains := lb.getOutboundFilterChainPerUpstream()

	listener := &xds_listener.Listener{
		Name:             outboundListenerName,
		Address:          envoy.GetAddress(constants.WildcardIPAddr, constants.EnvoyOutboundListenerPort),
		TrafficDirection: xds_core.TrafficDirection_OUTBOUND,
		FilterChains:     serviceFilterChains,
		ListenerFilters: []*xds_listener.ListenerFilter{
			{
				// The OriginalDestination ListenerFilter is used to redirect traffic
				// to its original destination.
				Name: wellknown.OriginalDestination,
			},
		},
		AccessLog: envoy.GetAccessLog(),
	}

	// Create a default passthrough filter chain when global egress is enabled.
	// This filter chain matches any traffic not matching any of the filter chains built from
	// mesh (SMI or permissive mode) or egress traffic policies. Traffic matching this default
	// passthrough filter chain will be allowed to passthrough to its original destination.
	if lb.cfg.IsEgressEnabled() {
		egressFilterChain, err := getDefaultPassthroughFilterChain()
		if err != nil {
			log.Error().Err(err).Msgf("Error getting filter chain for Egress")
			return nil, err
		}
		listener.DefaultFilterChain = egressFilterChain
	}

	if featureflags := lb.cfg.GetFeatureFlags(); featureflags.EnableEgressPolicy {
		var trafficMatches []*trafficpolicy.TrafficMatch
		var filterDisableMatchPredicate *xds_listener.ListenerFilterChainMatchPredicate
		// Create filter chains for egress based on policies
		if egressTrafficPolicy, err := lb.meshCatalog.GetEgressTrafficPolicy(lb.serviceIdentity); err != nil {
			log.Error().Err(err).Msgf("Error retrieving egress policies for proxy with identity %s, skipping egress filters", lb.serviceIdentity)
		} else if egressTrafficPolicy != nil {
			egressFilterChains := lb.getEgressFilterChainsForMatches(egressTrafficPolicy.TrafficMatches)
			listener.FilterChains = append(listener.FilterChains, egressFilterChains...)
			trafficMatches = append(trafficMatches, egressTrafficPolicy.TrafficMatches...)
		}
		trafficMatches = append(trafficMatches, lb.meshCatalog.GetOutboundMeshTrafficPolicy(lb.serviceIdentity).TrafficMatches...)
		filterDisableMatchPredicate = getFilterMatchPredicateForTrafficMatches(trafficMatches)
		additionalListenerFilters := []*xds_listener.ListenerFilter{
			// Configure match predicate for ports serving server-first protocols (ex. mySQL, postgreSQL etc.).
			// Ports corresponding to server-first protocols, where the server initiates the first byte of a connection, will
			// cause the HttpInspector ListenerFilter to timeout because it waits for data from the client to inspect the protocol.
			// Such ports will set the protocol to 'tcp-server-first' in an Egress policy.
			// The 'FilterDisabled' field configures the match predicate.
			{
				// To inspect TLS metadata, such as the transport protocol and SNI
				Name:           wellknown.TlsInspector,
				FilterDisabled: filterDisableMatchPredicate,
			},
			{
				// To inspect if the application protocol is HTTP based
				Name:           wellknown.HttpInspector,
				FilterDisabled: filterDisableMatchPredicate,
			},
		}
		listener.ListenerFilters = append(listener.ListenerFilters, additionalListenerFilters...)

		// ListenerFilter can timeout for server-first protocols. In such cases, continue the processing of the connection
		// and fallback to the default filter chain.
		listener.ContinueOnListenerFiltersTimeout = true
	}

	if len(listener.FilterChains) == 0 && listener.DefaultFilterChain == nil {
		// Programming a listener with no filter chains is an error.
		// It is possible for the outbound listener to have no filter chains if
		// there are no allowed upstreams for this proxy and egress is disabled.
		// In this case, return a nil filter chain so that it doesn't get programmed.
		return nil, nil
	}

	return listener, nil
}

func newInboundListener() *xds_listener.Listener {
	return &xds_listener.Listener{
		Name:             inboundListenerName,
		Address:          envoy.GetAddress(constants.WildcardIPAddr, constants.EnvoyInboundListenerPort),
		TrafficDirection: xds_core.TrafficDirection_INBOUND,
		FilterChains:     []*xds_listener.FilterChain{},
		ListenerFilters: []*xds_listener.ListenerFilter{
			{
				Name: wellknown.TlsInspector,
			},
			{
				// The OriginalDestination ListenerFilter is used to restore the original destination address
				// as opposed to the listener's address upon iptables redirection.
				// This enables inbound filter chain matching on the original destination address (ip, port).
				Name: wellknown.OriginalDestination,
			},
		},
		AccessLog: envoy.GetAccessLog(),
	}
}

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
						Name: wellknown.HTTPConnectionManager,
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
func getDefaultPassthroughFilterChain() (*xds_listener.FilterChain, error) {
	tcpProxy := &xds_tcp_proxy.TcpProxy{
		StatPrefix:       fmt.Sprintf("%s.%s", egressTCPProxyStatPrefix, envoy.OutboundPassthroughCluster),
		ClusterSpecifier: &xds_tcp_proxy.TcpProxy_Cluster{Cluster: envoy.OutboundPassthroughCluster},
	}
	marshalledTCPProxy, err := anypb.New(tcpProxy)
	if err != nil {
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrMarshallingXDSResource)).
			Msgf("Error marshalling TcpProxy object for egress HTTPS filter chain")
		return nil, err
	}

	return &xds_listener.FilterChain{
		Name: outboundEgressFilterChainName,
		Filters: []*xds_listener.Filter{
			{
				Name:       wellknown.TCPProxy,
				ConfigType: &xds_listener.Filter_TypedConfig{TypedConfig: marshalledTCPProxy},
			},
		},
	}, nil
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
