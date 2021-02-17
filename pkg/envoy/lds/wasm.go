package lds

import (
	"path/filepath"

	envoy_config_core_v3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	xds_wasm "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/wasm/v3"
	xds_hcm "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	xds_wasm_ext "github.com/envoyproxy/go-control-plane/envoy/extensions/wasm/v3"

	"github.com/golang/protobuf/ptypes"

	"github.com/openservicemesh/osm/pkg/constants"
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
