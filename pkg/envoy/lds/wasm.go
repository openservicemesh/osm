package lds

import (
	_ "embed" // required to embed resources
	"fmt"
	"strings"

	envoy_config_accesslog_v3 "github.com/envoyproxy/go-control-plane/envoy/config/accesslog/v3"
	envoy_config_core_v3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	xds_lua "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/lua/v3"
	xds_wasm "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/wasm/v3"
	xds_hcm "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	xds_wasm_ext "github.com/envoyproxy/go-control-plane/envoy/extensions/wasm/v3"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"github.com/pkg/errors"

	"google.golang.org/protobuf/types/known/anypb"
)

//go:embed stats.wasm
var statsWASMBytes []byte

func (lb *listenerBuilder) getWASMStatsHeaders() map[string]string {
	if lb.cfg.GetFeatureFlags().EnableWASMStats {
		return lb.statsHeaders
	}

	return nil
}

func getWASMStatsConfig(statsHeaders map[string]string) ([]*xds_hcm.HttpFilter, *xds_hcm.LocalReplyConfig, error) {
	statsFilter, err := getStatsWASMFilter()
	if err != nil {
		return nil, nil, errors.Wrap(err, "Error gettings WASM Stats filter")
	}

	headerFilter, err := getAddHeadersFilter(statsHeaders)
	if err != nil {
		return nil, nil, errors.Wrap(err, "Error getting WASM stats Header filter")
	}

	var filters []*xds_hcm.HttpFilter
	var localReplyConfig *xds_hcm.LocalReplyConfig
	if statsFilter != nil {
		if headerFilter != nil {
			filters = append(filters, headerFilter)
		}
		filters = append(filters, statsFilter)

		// When Envoy responds to an outgoing HTTP request with a local reply,
		// destination_* tags for WASM metrics are missing. This configures
		// Envoy's local replies to add the same headers that are expected from
		// HTTP responses with the "unknown" value hardcoded because we don't
		// know the intended destination of the request.
		var localReplyHeaders []*envoy_config_core_v3.HeaderValueOption
		for k := range statsHeaders {
			localReplyHeaders = append(localReplyHeaders, &envoy_config_core_v3.HeaderValueOption{
				Header: &envoy_config_core_v3.HeaderValue{
					Key:   k,
					Value: "unknown",
				},
			})
		}
		if localReplyHeaders != nil {
			localReplyConfig = &xds_hcm.LocalReplyConfig{
				Mappers: []*xds_hcm.ResponseMapper{
					{
						Filter: &envoy_config_accesslog_v3.AccessLogFilter{
							FilterSpecifier: &envoy_config_accesslog_v3.AccessLogFilter_NotHealthCheckFilter{},
						},
						HeadersToAdd: localReplyHeaders,
					},
				},
			}
		}
	}

	return filters, localReplyConfig, nil
}

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

	luaAny, err := anypb.New(lua)
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

	wasmAny, err := anypb.New(wasmPlug)
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
