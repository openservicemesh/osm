package lds

import (
	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	envoy_route "github.com/envoyproxy/go-control-plane/envoy/api/v2/route"
	envoy_core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	envoy_hcm "github.com/envoyproxy/go-control-plane/envoy/config/filter/network/http_connection_manager/v2"
	envoy_wasm "github.com/envoyproxy/go-control-plane/envoy/extensions/wasm/v3"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/wrappers"

	"github.com/open-service-mesh/osm/pkg/constants"
	"github.com/open-service-mesh/osm/pkg/envoy"
)

const (
	statPrefix = "http"
)

// TODO(draychev): move to OSM Config CRD or CLI
const (
	enableTracing = true
)

func getHTTPConnectionManager(routeName string) *envoy_hcm.HttpConnectionManager {
	wasm := &envoy_wasm.WasmService{
		Config: &envoy_wasm.PluginConfig{
			Name: "stats",
			VmConfig: &envoy_wasm.PluginConfig_InlineVmConfig{
				InlineVmConfig: &envoy_wasm.VmConfig{
					Runtime: "envoy.wasm.runtime.v8",
					Code: &envoy_core.AsyncDataSource{
						Specifier: &envoy_core.AsyncDataSource_Local{
							Local: &envoy_core.DataSource{
								Specifier: &envoy_core.DataSource_Filename{
									Filename: "/etc/envoy/stats.wasm",
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
		return nil
	}

	connManager := envoy_hcm.HttpConnectionManager{
		StatPrefix: statPrefix,
		CodecType:  envoy_hcm.HttpConnectionManager_AUTO,
		HttpFilters: []*envoy_hcm.HttpFilter{
			{
				Name: "envoy.filters.http.wasm",
				ConfigType: &envoy_hcm.HttpFilter_TypedConfig{
					TypedConfig: wasmAny,
				},
			},
			{
				Name: wellknown.Router,
			},
		},
		RouteSpecifier: &envoy_hcm.HttpConnectionManager_Rds{
			Rds: &envoy_hcm.Rds{
				ConfigSource:    envoy.GetADSConfigSource(),
				RouteConfigName: routeName,
			},
		},
		AccessLog: envoy.GetAccessLog(),
	}

	if enableTracing {
		connManager.GenerateRequestId = &wrappers.BoolValue{
			Value: true,
		}
		connManager.Tracing = &envoy_hcm.HttpConnectionManager_Tracing{
			Verbose: true,
		}
	}

	return &connManager
}

func getPrometheusConnectionManager(listenerName string, routeName string, clusterName string) *envoy_hcm.HttpConnectionManager {
	return &envoy_hcm.HttpConnectionManager{
		StatPrefix: listenerName,
		CodecType:  envoy_hcm.HttpConnectionManager_AUTO,
		HttpFilters: []*envoy_hcm.HttpFilter{{
			Name: wellknown.Router,
		}},
		RouteSpecifier: &envoy_hcm.HttpConnectionManager_RouteConfig{
			RouteConfig: &v2.RouteConfiguration{
				VirtualHosts: []*envoy_route.VirtualHost{{
					Name:    "prometheus_envoy_admin",
					Domains: []string{"*"},
					Routes: []*envoy_route.Route{{
						Match: &envoy_route.RouteMatch{
							PathSpecifier: &envoy_route.RouteMatch_Prefix{
								Prefix: routeName,
							},
						},
						Action: &envoy_route.Route_Route{
							Route: &envoy_route.RouteAction{
								ClusterSpecifier: &envoy_route.RouteAction_Cluster{
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
