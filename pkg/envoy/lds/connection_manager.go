package lds

import (
	"path/filepath"

	envoy_config_core_v3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	xds_route "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	xds_wasm "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/wasm/v3"
	xds_hcm "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	xds_wasm_ext "github.com/envoyproxy/go-control-plane/envoy/extensions/wasm/v3"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"

	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/wrappers"

	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/featureflags"
)

const (
	statPrefix = "http"
)

func getStatsWASMFilter() (*xds_hcm.HttpFilter, error) {
	wasm := &xds_wasm.Wasm{
		Config: &xds_wasm_ext.PluginConfig{
			Name: "stats",
			Vm: &xds_wasm_ext.PluginConfig_VmConfig{
				VmConfig: &xds_wasm_ext.VmConfig{
					Runtime: "envoy.wasm.runtime.v8",
					Code: &envoy_config_core_v3.AsyncDataSource{
						Specifier: &envoy_config_core_v3.AsyncDataSource_Local{
							Local: &envoy_config_core_v3.DataSource{
								Specifier: &envoy_config_core_v3.DataSource_Filename{
									Filename: filepath.Join(constants.StatsWASMLocation, constants.StatsWASMFilename),
								},
							},
						},
					},
					AllowPrecompiled: true,
				},
			},
		},
	}

	wasmAny, err := ptypes.MarshalAny(wasm)
	if err != nil {
		log.Error().Err(err).Msg("Error marshalling WasmService object")
		return nil, err
	}

	return &xds_hcm.HttpFilter{
		Name: "envoy.filters.http.wasm",
		ConfigType: &xds_hcm.HttpFilter_TypedConfig{
			TypedConfig: wasmAny,
		},
	}, nil
}

func getHTTPConnectionManager(routeName string, cfg configurator.Configurator) *xds_hcm.HttpConnectionManager {
	connManager := &xds_hcm.HttpConnectionManager{
		StatPrefix: statPrefix,
		CodecType:  xds_hcm.HttpConnectionManager_AUTO,
		HttpFilters: []*xds_hcm.HttpFilter{{
			Name: wellknown.Router,
		}},

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

		// wellknown.Router filter must be last
		var filters []*xds_hcm.HttpFilter
		filters = append(filters, statsFilter)
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
