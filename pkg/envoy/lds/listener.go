package lds

import (
	"net"
	"strconv"
	"strings"

	xds_core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	xds_listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	xds_hcm "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	xds_tcp_proxy "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/tcp_proxy/v3"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/golang/protobuf/ptypes/wrappers"

	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/envoy/route"
	"github.com/openservicemesh/osm/pkg/kubernetes"
	"github.com/openservicemesh/osm/pkg/service"
)

const (
	outboundMeshFilterChainName   = "outbound-mesh-filter-chain"
	outboundEgressFilterChainName = "outbound-egress-filter-chain"
	singleIpv4Mask                = 32
)

func newOutboundListener(catalog catalog.MeshCataloger, cfg configurator.Configurator, localSvc []service.MeshService) (*xds_listener.Listener, error) {
	outboundListener := &xds_listener.Listener{
		Name:             outboundListenerName,
		Address:          envoy.GetAddress(constants.WildcardIPAddr, constants.EnvoyOutboundListenerPort),
		TrafficDirection: xds_core.TrafficDirection_OUTBOUND,
		FilterChains:     []*xds_listener.FilterChain{},
		ListenerFilters: []*xds_listener.ListenerFilter{
			{
				// The OriginalDestination ListenerFilter is used to redirect traffic
				// to its original destination.
				Name: wellknown.OriginalDestination,
			},
		},
	}

	serviceFilterChains, err := getFilterChains(catalog, cfg, localSvc)
	if err != nil {
		log.Error().Err(err).Msgf("Error getting filter chain for Egress")
		return nil, err
	}

	outboundListener.FilterChains = append(outboundListener.FilterChains, serviceFilterChains...)
	return outboundListener, nil
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
	marshalledTCPProxy, err := envoy.MessageToAny(tcpProxy)
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

func parseCIDR(cidr string) (string, uint32, error) {
	var addr string

	_, _, err := net.ParseCIDR(cidr)
	if err != nil {
		return addr, 0, err
	}
	chunks := strings.Split(cidr, "/")
	addr = chunks[0]
	prefix, err := strconv.Atoi(chunks[1])
	if err != nil {
		return addr, 0, err
	}

	return addr, uint32(prefix), nil
}

func getCIDRRange(cidr string) (*xds_core.CidrRange, error) {
	addr, prefix, err := parseCIDR(cidr)
	if err != nil {
		return nil, err
	}

	cidrRange := &xds_core.CidrRange{
		AddressPrefix: addr,
		PrefixLen: &wrappers.UInt32Value{
			Value: prefix,
		},
	}

	return cidrRange, nil
}

// getFilterForService builds a network filter action for traffic destined to a specific service
func getFilterForService(meshSvc service.MeshService, cfg configurator.Configurator) (string, *any.Any, error) {
	var marshalledFilter *any.Any
	var wellknownFilterName string
	var err error

	if cfg.IsPermissiveTrafficPolicyMode() {
		// Stright TCP proxy
		wellknownFilterName = wellknown.TCPProxy

		tcpProxy := &xds_tcp_proxy.TcpProxy{
			StatPrefix: meshSvc.String(),
			ClusterSpecifier: &xds_tcp_proxy.TcpProxy_Cluster{
				// TODO: Move this to get Cluster name based on MeshService
				Cluster: meshSvc.String(),
			},
		}

		marshalledFilter, err = envoy.MessageToAny(tcpProxy)
		if err != nil {
			log.Error().Err(err).Msgf("Error marshalling TcpProxy object")
			return "", nil, err
		}
	} else {
		// Have HTTP Connection Manager (and later RDS) do the policying
		wellknownFilterName = wellknown.HTTPConnectionManager

		marshalledFilter, err = envoy.MessageToAny(
			getHTTPConnectionManager(route.OutboundRouteConfigName, cfg))
		if err != nil {
			log.Error().Err(err).Msgf("Error marshalling HTTPConnManager object")
			return "", nil, err
		}
	}

	return wellknownFilterName, marshalledFilter, nil
}

// getFilterChainMatchForService builds a filter chain to match the destination traffic.
// Filter Chain matches currently match on destination IP for possible service endpoint leafs
func getFilterChainMatchForService(meshSvc service.MeshService, catalog catalog.MeshCataloger, cfg configurator.Configurator) (*xds_listener.FilterChainMatch, error) {
	ret := &xds_listener.FilterChainMatch{}

	endpoints, err := catalog.GetResolvableServiceEndpoints(meshSvc)
	if err != nil {
		log.Error().Err(err).Msgf("Error getting GetResolvableServiceEndpoints for %s", meshSvc.String())
		return nil, err
	}

	if len(endpoints) == 0 {
		log.Error().Err(err).Msgf("No resolvable addresses retured for service %s", meshSvc.String())
		return nil, errNoValidTargetEndpoints
	}

	for _, endp := range endpoints {
		ret.PrefixRanges = append(ret.PrefixRanges, &xds_core.CidrRange{
			AddressPrefix: endp.IP.String(),
			PrefixLen: &wrapperspb.UInt32Value{
				Value: singleIpv4Mask,
			},
		})
	}

	return ret, nil
}

func getFilterChains(catalog catalog.MeshCataloger, cfg configurator.Configurator, localSvc []service.MeshService) ([]*xds_listener.FilterChain, error) {
	var ret []*xds_listener.FilterChain
	var dstServicesSet map[service.MeshService]struct{} = make(map[service.MeshService]struct{}) // Set, avoid dups

	outboundSvc, err := catalog.ListAllowedOutboundServices(localSvc[0])
	if err != nil {
		log.Error().Err(err).Msgf("Error getting allowed outbound services for %s", localSvc[0].String())
		return nil, err
	}

	// Transform into set, when listing apex services we might face repetitions
	for _, meshSvc := range outboundSvc {
		dstServicesSet[meshSvc] = struct{}{}
	}

	// Getting apex services referring to the outbound services
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

		// Filterchain for current dest
		filterChain := xds_listener.FilterChain{
			Name:             keyService.String(),
			Filters:          []*xds_listener.Filter{},
			FilterChainMatch: &xds_listener.FilterChainMatch{},
		}

		// Get filter for service
		filterName, filterProto, err := getFilterForService(keyService, cfg)
		if err != nil {
			log.Error().Err(err).Msgf("Error getting filter for dst service %s", keyService.String())
			return nil, err
		}
		filterChain.Filters = append(filterChain.Filters, &xds_listener.Filter{
			Name:       filterName,
			ConfigType: &xds_listener.Filter_TypedConfig{TypedConfig: filterProto},
		})

		// Get filter match criteria for destination service
		filterChainMatch, err := getFilterChainMatchForService(keyService, catalog, cfg)
		if err != nil {
			log.Error().Err(err).Msgf("Error getting Chain Match for service %s", keyService.String())
			return nil, err
		}

		filterChain.FilterChainMatch = filterChainMatch
		ret = append(ret, &filterChain)
	}

	// This filterchain represents connectivity to any cluster not explicitely part of the mesh,
	// and can be used by both permissive and egress
	if cfg.IsEgressEnabled() || cfg.IsPermissiveTrafficPolicyMode() {
		egressFilterChgain, err := buildEgressFilterChain()
		if err != nil {
			log.Error().Err(err).Msgf("Error getting filter chain for Egress")
			return nil, err
		}

		ret = append(ret, egressFilterChgain)
	}

	return ret, nil
}
