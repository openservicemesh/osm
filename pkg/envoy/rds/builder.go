package rds

import (
	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	xds_route "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"

	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/trafficpolicy"
)

const (
	// inboundVirtualHost is prefix for the virtual host's name in the inbound route configuration
	inboundVirtualHost = "inbound_virtual-host"

	// outboundVirtualHost is the prefix for the virtual host's name in the outbound route configuration
	outboundVirtualHost = "outbound_virtual-host"

	// egressVirtualHost is the prefix for the virtual host's name in the egress route configuration
	egressVirtualHost = "egress_virtual-host"

	// ingressVirtualHost is the prefix for the virtual host's name in the ingress route configuration
	ingressVirtualHost = "ingress_virtual-host"
)

type routesBuilder struct {
	inboundPortSpecificRouteConfigs  map[int][]*trafficpolicy.InboundTrafficPolicy
	outboundPortSpecificRouteConfigs map[int][]*trafficpolicy.OutboundTrafficPolicy
	ingressTrafficPolicies           []*trafficpolicy.InboundTrafficPolicy
	egressPortSpecificRouteConfigs   map[int][]*trafficpolicy.EgressHTTPRouteConfig
	proxy                            *envoy.Proxy
	statsHeaders                     map[string]string
	trustDomain                      string
}

func RoutesBuilder() *routesBuilder { //nolint: revive // unexported-return
	return &routesBuilder{}
}

func (b *routesBuilder) InboundPortSpecificRouteConfigs(inboundPortSpecificRouteConfigs map[int][]*trafficpolicy.InboundTrafficPolicy) *routesBuilder {
	b.inboundPortSpecificRouteConfigs = inboundPortSpecificRouteConfigs
	return b
}

func (b *routesBuilder) OutboundPortSpecificRouteConfigs(outboundPortSpecificRouteConfigs map[int][]*trafficpolicy.OutboundTrafficPolicy) *routesBuilder {
	b.outboundPortSpecificRouteConfigs = outboundPortSpecificRouteConfigs
	return b
}

func (b *routesBuilder) IngressTrafficPolicies(ingressTrafficPolicies []*trafficpolicy.InboundTrafficPolicy) *routesBuilder {
	b.ingressTrafficPolicies = ingressTrafficPolicies
	return b
}

func (b *routesBuilder) EgressPortSpecificRouteConfigs(egressPortSpecificRouteConfigs map[int][]*trafficpolicy.EgressHTTPRouteConfig) *routesBuilder {
	b.egressPortSpecificRouteConfigs = egressPortSpecificRouteConfigs
	return b
}

func (b *routesBuilder) Proxy(proxy *envoy.Proxy) *routesBuilder {
	b.proxy = proxy
	return b
}

func (b *routesBuilder) StatsHeaders(statsHeaders map[string]string) *routesBuilder {
	b.statsHeaders = statsHeaders
	return b
}

func (b *routesBuilder) TrustDomain(trustDomain string) *routesBuilder {
	b.trustDomain = trustDomain
	return b
}

// buildInboundMeshRouteConfiguration constructs the Envoy constructs ([]*xds_route.RouteConfiguration) for implementing inbound and outbound routes
func (b *routesBuilder) buildInboundMeshRouteConfiguration() []*xds_route.RouteConfiguration {
	var routeConfigs []*xds_route.RouteConfiguration

	// An Envoy RouteConfiguration will exist for each HTTP upstream port.
	// This is required to avoid route conflicts that can arise when the same host header
	// has different routes on different destination ports for that host.
	for port, configs := range b.inboundPortSpecificRouteConfigs {
		routeConfig := newRouteConfigurationStub(GetInboundMeshRouteConfigNameForPort(port))
		for _, config := range configs {
			virtualHost := buildVirtualHostStub(inboundVirtualHost, config.Name, config.Hostnames)
			virtualHost.Routes = buildInboundRoutes(config.Rules, b.trustDomain)
			applyInboundVirtualHostConfig(virtualHost, config)
			routeConfig.VirtualHosts = append(routeConfig.VirtualHosts, virtualHost)
		}
		for k, v := range b.statsHeaders {
			routeConfig.ResponseHeadersToAdd = append(routeConfig.ResponseHeadersToAdd, &core.HeaderValueOption{
				Header: &core.HeaderValue{
					Key:   k,
					Value: v,
				},
			})
		}
		routeConfigs = append(routeConfigs, routeConfig)
	}

	return routeConfigs
}

// buildIngressConfiguration constructs the Envoy constructs ([]*xds_route.RouteConfiguration) for implementing ingress routes
func (b *routesBuilder) buildIngressConfiguration() *xds_route.RouteConfiguration {
	if len(b.ingressTrafficPolicies) == 0 {
		return nil
	}

	ingressRouteConfig := newRouteConfigurationStub(IngressRouteConfigName)
	for _, in := range b.ingressTrafficPolicies {
		virtualHost := buildVirtualHostStub(ingressVirtualHost, in.Name, in.Hostnames)
		virtualHost.Routes = buildInboundRoutes(in.Rules, b.trustDomain)
		ingressRouteConfig.VirtualHosts = append(ingressRouteConfig.VirtualHosts, virtualHost)
	}

	return ingressRouteConfig
}

// buildOutboundMeshRouteConfiguration constructs the Envoy construct (*xds_route.RouteConfiguration) for the given outbound mesh route configs
func (b *routesBuilder) buildOutboundMeshRouteConfiguration() []*xds_route.RouteConfiguration {
	var routeConfigs []*xds_route.RouteConfiguration

	// An Envoy RouteConfiguration will exist for each HTTP upstream port.
	// This is required to avoid route conflicts that can arise when the same host header
	// has different routes on different destination ports for that host.
	for port, configs := range b.outboundPortSpecificRouteConfigs {
		routeConfig := newRouteConfigurationStub(GetOutboundMeshRouteConfigNameForPort(port))
		for _, config := range configs {
			virtualHost := buildVirtualHostStub(outboundVirtualHost, config.Name, config.Hostnames)
			virtualHost.Routes = buildOutboundRoutes(config.Routes)
			routeConfig.VirtualHosts = append(routeConfig.VirtualHosts, virtualHost)
		}
		routeConfigs = append(routeConfigs, routeConfig)
	}

	return routeConfigs
}

// buildEgressRouteConfiguration constructs the Envoy construct (*xds_route.RouteConfiguration) for the given egress route configs
func (b *routesBuilder) buildEgressRouteConfiguration() []*xds_route.RouteConfiguration {
	var routeConfigs []*xds_route.RouteConfiguration

	// An Envoy RouteConfiguration will exist for each HTTP egress port.
	// This is required to avoid route conflicts that can arise when the same host header
	// has different routes on different destination ports for that host.
	for port, configs := range b.egressPortSpecificRouteConfigs {
		routeConfig := newRouteConfigurationStub(GetEgressRouteConfigNameForPort(port))
		for _, config := range configs {
			virtualHost := buildVirtualHostStub(egressVirtualHost, config.Name, config.Hostnames)
			virtualHost.Routes = buildEgressRoutes(config.RoutingRules)
			routeConfig.VirtualHosts = append(routeConfig.VirtualHosts, virtualHost)
		}
		routeConfigs = append(routeConfigs, routeConfig)
	}

	return routeConfigs
}

func (b *routesBuilder) Build() ([]types.Resource, error) {
	var rdsResources []types.Resource

	// ---
	// Build inbound mesh route configurations. These route configurations allow
	// the services associated with this proxy to accept traffic from downstream
	// clients on allowed routes.
	if b.inboundPortSpecificRouteConfigs != nil {
		inboundMeshRouteConfig := b.buildInboundMeshRouteConfiguration()
		for _, config := range inboundMeshRouteConfig {
			rdsResources = append(rdsResources, config)
		}
	}

	// ---
	// Build outbound mesh route configurations. These route configurations allow this proxy
	// to direct traffic to upstream services that it is authorized to connect to on allowed
	// routes.
	if b.outboundPortSpecificRouteConfigs != nil {
		outboundMeshRouteConfig := b.buildOutboundMeshRouteConfiguration()
		for _, config := range outboundMeshRouteConfig {
			rdsResources = append(rdsResources, config)
		}
	}

	// ---
	// Build ingress route configurations. These route configurations allow the
	// services associated with this proxy to accept ingress traffic from downstream
	// clients on allowed routes.
	if len(b.ingressTrafficPolicies) > 0 {
		ingressRouteConfig := b.buildIngressConfiguration()
		rdsResources = append(rdsResources, ingressRouteConfig)
	}

	// ---
	// Build egress route configurations. These route configurations allow this
	// proxy to direct traffic to external non-mesh destinations on allowed routes.
	if b.egressPortSpecificRouteConfigs != nil {
		egressRouteConfigs := b.buildEgressRouteConfiguration()
		for _, egressConfig := range egressRouteConfigs {
			rdsResources = append(rdsResources, egressConfig)
		}
	}

	return rdsResources, nil
}
