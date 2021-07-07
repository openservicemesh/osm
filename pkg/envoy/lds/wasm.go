package lds

import (
	_ "embed" // required to embed resources
	"fmt"
	"strings"

	envoy_config_core_v3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	envoy_config_listener_v3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	xds_lua "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/lua/v3"
	xds_wasm "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/wasm/v3"
	xds_hcm "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	xds_network_wasm "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/wasm/v3"
	xds_wasm_ext "github.com/envoyproxy/go-control-plane/envoy/extensions/wasm/v3"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"github.com/pkg/errors"

	"github.com/golang/protobuf/ptypes"
)

//go:embed stats.wasm
var statsWASMBytes []byte

func getAddHeadersFilter(headers map[string]string) (*xds_hcm.HttpFilter, error) {
	if len(headers) == 0 {
		return nil, nil
	}
	addCallsReq := &strings.Builder{}
	addCallsReq.WriteString("--\nfunction envoy_on_request(request_handle)\n")
	for k, v := range headers {
		addCallsReq.WriteString(fmt.Sprintf("  request_handle:headers():add(%q, %q)\n", k, v))
	}
	addCallsReq.WriteString("end")

	lua := &xds_lua.Lua{
		InlineCode: addCallsReq.String(),
	}

	luaAny, err := ptypes.MarshalAny(lua)
	if err != nil {
		return nil, errors.Wrap(err, "error marshaling Lua filter")
	}

	return &xds_hcm.HttpFilter{
		Name: wellknown.Lua,
		ConfigType: &xds_hcm.HttpFilter_TypedConfig{
			TypedConfig: luaAny,
		},
	}, nil
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
									InlineBytes: statsWASMBytes,
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
		return nil, errors.Wrap(err, "Error marshalling Wasm config")
	}

	return &xds_hcm.HttpFilter{
		Name: "envoy.filters.http.wasm",
		ConfigType: &xds_hcm.HttpFilter_TypedConfig{
			TypedConfig: wasmAny,
		},
	}, nil
}

func getStatsNetworkWASMFilter() (*envoy_config_listener_v3.ListenerFilter, error) {
	if len(statsWASMBytes) == 0 {
		return nil, nil
	}
	wasmPlug := &xds_network_wasm.Wasm{
		Config: &xds_wasm_ext.PluginConfig{
			Name: "stats",
			Vm: &xds_wasm_ext.PluginConfig_VmConfig{
				VmConfig: &xds_wasm_ext.VmConfig{
					Runtime: "envoy.wasm.runtime.v8",
					Code: &envoy_config_core_v3.AsyncDataSource{
						Specifier: &envoy_config_core_v3.AsyncDataSource_Local{
							Local: &envoy_config_core_v3.DataSource{
								Specifier: &envoy_config_core_v3.DataSource_InlineBytes{
									InlineBytes: statsWASMBytes,
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
		return nil, errors.Wrap(err, "Error marshalling Wasm config")
	}

	return &envoy_config_listener_v3.ListenerFilter{
		Name: "envoy.filters.network.wasm",
		ConfigType: &envoy_config_listener_v3.ListenerFilter_TypedConfig{
			TypedConfig: wasmAny,
		},
	}, nil
}