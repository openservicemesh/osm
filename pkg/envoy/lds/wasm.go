package lds

import (
	"encoding/base64"
	"io/ioutil"
	"strings"

	envoy_config_core_v3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	xds_wasm "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/wasm/v3"
	xds_hcm "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	xds_wasm_ext "github.com/envoyproxy/go-control-plane/envoy/extensions/wasm/v3"

	"github.com/golang/protobuf/ptypes"
)

var statsWASMBytes string

func init() {
	b64 := base64.NewDecoder(base64.StdEncoding, strings.NewReader(statsWASMBytes))
	b, _ := ioutil.ReadAll(b64)
	statsWASMBytes = string(b)
}

func getStatsWASMFilter() (*xds_hcm.HttpFilter, error) {
	if len(statsWASMBytes) == 0 {
		return nil, nil
	}
	wasmPlug := &xds_wasm.Wasm{
		Config: &xds_wasm_ext.PluginConfig{
			Name: "stats",
			Vm: &xds_wasm_ext.PluginConfig_VmConfig{
				VmConfig: &xds_wasm_ext.VmConfig{
					Runtime: "envoy.wasm.runtime.v8",
					Code: &envoy_config_core_v3.AsyncDataSource{
						Specifier: &envoy_config_core_v3.AsyncDataSource_Local{
							Local: &envoy_config_core_v3.DataSource{
								Specifier: &envoy_config_core_v3.DataSource_InlineBytes{
									InlineBytes: []byte(statsWASMBytes),
								},
							},
						},
					},
					AllowPrecompiled: true,
				},
			},
		},
	}

	wasmAny, err := ptypes.MarshalAny(wasmPlug)
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
