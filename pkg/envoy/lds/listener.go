package lds

import (
	xds_core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	xds_listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	xds_hcm "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	xds_tcp_proxy "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/tcp_proxy/v3"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"github.com/golang/protobuf/ptypes"

	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/kubernetes"
	"github.com/openservicemesh/osm/pkg/service"
)

const (
	outboundEgressFilterChainName = "outbound-egress-filter-chain"
	singleIpv4Mask                = 32
)

func newOutboundListener(catalog catalog.MeshCataloger, cfg configurator.Configurator, downstreamSvc []service.MeshService) (*xds_listener.Listener, error) {
	serviceFilterChains, err := getOutboundFilterChains(catalog, cfg, downstreamSvc)
	if err != nil {
		log.Error().Err(err).Msgf("Error getting filter chains for outbound listener")
		return nil, err
	}

	if len(serviceFilterChains) == 0 {
		log.Info().Msgf("No filterchains for outbound services. Not programming Outbound listener.")
		return nil, nil
	}

	return &xds_listener.Listener{
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
			{
				// The HttpInspector ListenerFilter is used to inspect plaintext traffic
				// for HTTP protocols.
				Name: wellknown.HttpInspector,
			},
		},
	}, nil
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
		},
	}
}

func buildPrometheusListener(connManager *xds_hcm.HttpConnectionManager) (*xds_listener.Listener, error) {
	marshalledConnManager, err := ptypes.MarshalAny(connManager)
	if err != nil {
		log.Error().Err(err).Msgf("Error marshalling HttpConnectionManager object")
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

func buildEgressFilterChain() (*xds_listener.FilterChain, error) {
	tcpProxy := &xds_tcp_proxy.TcpProxy{
		StatPrefix:       envoy.OutboundPassthroughCluster,
		ClusterSpecifier: &xds_tcp_proxy.TcpProxy_Cluster{Cluster: envoy.OutboundPassthroughCluster},
	}
	marshalledTCPProxy, err := ptypes.MarshalAny(tcpProxy)
	if err != nil {
		log.Error().Err(err).Msgf("Error marshalling TcpProxy object for egress HTTPS filter chain")
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

func getOutboundFilterChains(catalog catalog.MeshCataloger, cfg configurator.Configurator, downstreamSvc []service.MeshService) ([]*xds_listener.FilterChain, error) {
	var filterChains []*xds_listener.FilterChain
	var dstServicesSet map[service.MeshService]struct{} = make(map[service.MeshService]struct{}) // Set, avoid dups

	// Assuming single service in pod till #1682, #1575 get addressed
	outboundSvc, err := catalog.ListAllowedOutboundServices(downstreamSvc[0])
	if err != nil {
		log.Error().Err(err).Msgf("Error getting allowed outbound services for %s", downstreamSvc[0].String())
		return nil, err
	}

	// Transform into set, when listing apex services we might face repetitions
	for _, meshSvc := range outboundSvc {
		dstServicesSet[meshSvc] = struct{}{}
	}

	// Getting apex services referring to the outbound services
	// We get possible apexes which could traffic split to any of the possible
	// outbound services
	splitServices := catalog.GetSMISpec().ListTrafficSplitServices()
	for _, svc := range splitServices {
		for _, outSvc := range outboundSvc {
			if svc.Service == outSvc {
				rootServiceName := kubernetes.GetServiceFromHostname(svc.RootService)
				rootMeshService := service.MeshService{
					Namespace: outSvc.Namespace,
					Name:      rootServiceName,
				}

				// Add this root service into the set
				dstServicesSet[rootMeshService] = struct{}{}
			}
		}
	}

	// Iterate all destination services
	for keyService := range dstServicesSet {
		// Get filter for service
		filter, err := getOutboundFilterForService(keyService, cfg)
		if err != nil {
			log.Error().Err(err).Msgf("Error getting filter for dst service %s", keyService.String())
			return nil, err
		}

		// Get filter match criteria for destination service
		filterChainMatch, err := getOutboundHTTPFilterChainMatchForService(keyService, catalog, cfg)
		if err != nil {
			log.Error().Err(err).Msgf("Error getting Chain Match for service %s", keyService.String())
			return nil, err
		}
		if filterChainMatch == nil {
			log.Info().Msgf("No endpoints found for dst service %s. Not adding filterchain.", keyService)
			continue
		}

		filterChains = append(filterChains, &xds_listener.FilterChain{
			Name:             keyService.String(),
			Filters:          []*xds_listener.Filter{filter},
			FilterChainMatch: filterChainMatch,
		})
	}

	// This filterchain matches any traffic not filtered by allow rules, it will be treated as egress
	// traffic when enabled
	if cfg.IsEgressEnabled() {
		egressFilterChgain, err := buildEgressFilterChain()
		if err != nil {
			log.Error().Err(err).Msgf("Error getting filter chain for Egress")
			return nil, err
		}

		filterChains = append(filterChains, egressFilterChgain)
	}

	return filterChains, nil
}
