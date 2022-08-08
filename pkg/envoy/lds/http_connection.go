package lds

import (
	xds_route "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	xds_hcm "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	"github.com/golang/protobuf/ptypes/any"

	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/envoy"
)

const (
	prometheusHTTPConnManagerStatPrefix = "prometheus-http-conn-manager"
	prometheusInboundVirtualHostName    = "prometheus-inbound-virtual-host"
)

func getPrometheusConnectionManager() *xds_hcm.HttpConnectionManager {
	return &xds_hcm.HttpConnectionManager{
		StatPrefix: prometheusHTTPConnManagerStatPrefix,
		CodecType:  xds_hcm.HttpConnectionManager_AUTO,
		HttpFilters: []*xds_hcm.HttpFilter{
			{
				Name: envoy.HTTPRouterFilterName,
				ConfigType: &xds_hcm.HttpFilter_TypedConfig{
					TypedConfig: &any.Any{
						TypeUrl: envoy.HTTPRouterFilterTypeURL,
					},
				},
			},
		},
		RouteSpecifier: &xds_hcm.HttpConnectionManager_RouteConfig{
			RouteConfig: &xds_route.RouteConfiguration{
				VirtualHosts: []*xds_route.VirtualHost{{
					Name:    prometheusInboundVirtualHostName,
					Domains: []string{"*"}, // Match all domains
					Routes: []*xds_route.Route{{
						Match: &xds_route.RouteMatch{
							PathSpecifier: &xds_route.RouteMatch_Prefix{
								Prefix: constants.PrometheusScrapePath,
							},
						},
						Action: &xds_route.Route_Route{
							Route: &xds_route.RouteAction{
								ClusterSpecifier: &xds_route.RouteAction_Cluster{
									Cluster: constants.EnvoyMetricsCluster,
								},
								PrefixRewrite: constants.PrometheusScrapePath,
							},
						},
					}},
				}},
			},
		},
		AccessLog: envoy.GetAccessLog(),
	}
}
