package lds

import (
	envoy_route "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	envoy_hcm "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/wrappers"
	"google.golang.org/protobuf/types/known/structpb"

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

	// Using untyped proto till the changes are propagated to the APIs upstream
	wasm := &structpb.Struct{
		Fields: map[string]*structpb.Value{
			"config": {
				Kind: &structpb.Value_StructValue{
					StructValue: &structpb.Struct{
						Fields: map[string]*structpb.Value{
							"name": {
								Kind: &structpb.Value_StringValue{
									StringValue: "stats",
								},
							},
							"vm_config": {
								Kind: &structpb.Value_StructValue{
									StructValue: &structpb.Struct{
										Fields: map[string]*structpb.Value{
											"runtime": {
												Kind: &structpb.Value_StringValue{
													StringValue: "envoy.wasm.runtime.v8",
												},
											},
											"code": {
												Kind: &structpb.Value_StructValue{
													StructValue: &structpb.Struct{
														Fields: map[string]*structpb.Value{
															"local": {
																Kind: &structpb.Value_StructValue{
																	StructValue: &structpb.Struct{
																		Fields: map[string]*structpb.Value{
																			"filename": {
																				Kind: &structpb.Value_StringValue{
																					// TODO: prob use configmap, right now the wasm filename comes in
																					// injector config, which is not easilly available in this scope
																					StringValue: constants.EnvoyWasmFileloc + "/" + "stats.wasm",
																				},
																			},
																		},
																	},
																},
															},
														},
													},
												},
											},
											"allow_precompiled": {
												Kind: &structpb.Value_BoolValue{BoolValue: true},
											},
										},
									},
								},
							},
						},
					},
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
			RouteConfig: &envoy_route.RouteConfiguration{
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
