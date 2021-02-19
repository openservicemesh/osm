package lds

import (
	xds_route "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	xds_hcm "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"

	"github.com/golang/protobuf/ptypes/wrappers"

	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/featureflags"
)

const (
	statPrefix = "http"
)

func getHTTPConnectionManager(routeName string, cfg configurator.Configurator, headers map[string]string) *xds_hcm.HttpConnectionManager {
	connManager := &xds_hcm.HttpConnectionManager{
		StatPrefix: statPrefix,
		CodecType:  xds_hcm.HttpConnectionManager_AUTO,
		HttpFilters: []*xds_hcm.HttpFilter{
			{
				// HTTP RBAC filter
				Name: wellknown.HTTPRoleBasedAccessControl,
			},
			{
				// HTTP Router filter
				Name: wellknown.Router,
			},
		},

		RouteSpecifier: &xds_hcm.HttpConnectionManager_Rds{
			Rds: &xds_hcm.Rds{
				ConfigSource:    envoy.GetADSConfigSource(),
				RouteConfigName: routeName,
			},
		},
		AccessLog: envoy.GetAccessLog(),
	}

	if cfg.IsTracingEnabled() {
		connManager.GenerateRequestId = &wrappers.BoolValue{
			Value: true,
		}

		tracing, err := GetTracingConfig(cfg)
		if err != nil {
			log.Error().Err(err).Msg("Error getting tracing config")
			return connManager
		}

		connManager.Tracing = tracing
	}

	if featureflags.IsWASMStatsEnabled() {
		statsFilter, err := getStatsWASMFilter()
		if err != nil {
			log.Error().Err(err).Msg("failed to get stats WASM filter")
			return connManager
		}

		headerFilter, err := getAddHeadersFilter(headers)
		if err != nil {
			log.Error().Err(err).Msg("Could not get Lua filter definition")
			return connManager
		}

		// wellknown.Router filter must be last
		var filters []*xds_hcm.HttpFilter
		if headerFilter != nil {
			filters = append(filters, headerFilter)
		}
		if statsFilter != nil {
			filters = append(filters, statsFilter)
		}
		connManager.HttpFilters = append(filters, connManager.HttpFilters...)
	}

	return connManager
}

func getPrometheusConnectionManager(listenerName string, routeName string, clusterName string) *xds_hcm.HttpConnectionManager {
	return &xds_hcm.HttpConnectionManager{
		StatPrefix: listenerName,
		CodecType:  xds_hcm.HttpConnectionManager_AUTO,
		HttpFilters: []*xds_hcm.HttpFilter{{
			Name: wellknown.Router,
		}},
		RouteSpecifier: &xds_hcm.HttpConnectionManager_RouteConfig{
			RouteConfig: &xds_route.RouteConfiguration{
				VirtualHosts: []*xds_route.VirtualHost{{
					Name:    "prometheus_envoy_admin",
					Domains: []string{"*"},
					Routes: []*xds_route.Route{{
						Match: &xds_route.RouteMatch{
							PathSpecifier: &xds_route.RouteMatch_Prefix{
								Prefix: routeName,
							},
						},
						Action: &xds_route.Route_Route{
							Route: &xds_route.RouteAction{
								ClusterSpecifier: &xds_route.RouteAction_Cluster{
									Cluster: clusterName,
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
