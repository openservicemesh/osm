package lds

import (
	"fmt"

	xds_listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/service"
)

func (lb *listenerBuilder) newMultiClusterGatewayListener() *xds_listener.Listener {
	return &xds_listener.Listener{
		Name:    multiclusterListenerName,
		Address: envoy.GetAddress(constants.WildcardIPAddr, constants.EnvoyInboundListenerPort),
		// TODO(steeling) for this to work on windows, there needs to be an inbound and an outbound listener
		// see: https://www.envoyproxy.io/docs/envoy/latest/configuration/listeners/listener_filters/original_dst_filter#config-listener-filters-original-dst
		// TrafficDirection: ...,
		FilterChains: lb.getMultiClusterGatewayFilterChainPerUpstream(),
		ListenerFilters: []*xds_listener.ListenerFilter{
			{
				// The OriginalDestination ListenerFilter is used to redirect traffic
				// to its original destination.
				Name: wellknown.OriginalDestination,
			},
		},
	}
}

func (lb *listenerBuilder) getMultiClusterGatewayFilterChainPerUpstream() []*xds_listener.FilterChain {
	var filterChains []*xds_listener.FilterChain

	dstServices := lb.meshCatalog.ListMeshServicesForIdentity(lb.serviceIdentity)
	if len(dstServices) == 0 {
		log.Debug().Msgf("Proxy with identity %s does not have any allowed upstream services", lb.serviceIdentity)
		return filterChains
	}

	// Iterate all destination services
	for _, upstream := range dstServices {
		// Filter out to only the local and global services.
		// TODO(steeling): local here needs to the remote name.
		if !upstream.Local() {
			continue
		}

		log.Trace().Msgf("Building outbound filter chain for upstream service %s for proxy with identity %s", upstream, lb.serviceIdentity)
		protocolToPortMap, err := lb.meshCatalog.GetPortToProtocolMappingForService(upstream)
		if err != nil {
			log.Error().Err(err).Msgf("Error retrieving port to protocol mapping for upstream service %s", upstream)
			continue
		}

		// Create protocol specific inbound filter chains per port to handle different ports serving different protocols
		for port := range protocolToPortMap {
			// The gateway uses SSL passthrough, so simply uses a TCP filter.
			filter, err := lb.getOutboundTCPFilter(upstream)
			if err != nil {
				log.Error().Err(err).Msgf("Error getting tcp filter for upstream service %s", upstream)
				continue
			}

			hostnames, _ := lb.meshCatalog.GetServiceHostnames(upstream, service.RemoteCluster)
			filterChains = append(filterChains, &xds_listener.FilterChain{
				Name:    fmt.Sprintf("%s:%s", outboundMeshTCPFilterChainPrefix, upstream),
				Filters: []*xds_listener.Filter{filter},
				FilterChainMatch: &xds_listener.FilterChainMatch{
					DestinationPort: &wrapperspb.UInt32Value{
						Value: port,
					},
					ServerNames:          hostnames,
					ApplicationProtocols: envoy.ALPNInMesh,
					TransportProtocol:    envoy.TransportProtocolTLS,
				},
			})
		}
	}

	return filterChains
}
