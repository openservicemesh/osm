package lds

import (
	"fmt"

	xds_route "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	xds_local_ratelimit "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/local_ratelimit/v3"
	xds_hcm "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/golang/protobuf/ptypes/wrappers"

	"github.com/openservicemesh/osm/pkg/auth"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/protobuf"
)

// connectionDirection defines, for filter terms, the direction of a connection from
// the proxy's perspective while originating/terminating connections to/from other
// proxies.
type connectionDirection string

const (
	meshHTTPConnManagerStatPrefix       = "mesh-http-conn-manager"
	prometheusHTTPConnManagerStatPrefix = "prometheus-http-conn-manager"
	prometheusInboundVirtualHostName    = "prometheus-inbound-virtual-host"
	websocketUpgradeType                = "websocket"

	// inbound defines in-mesh inbound or ingress traffic driections
	inbound connectionDirection = "inbound"

	// outbound defines in-mesh outbound or egress traffic directions
	outbound connectionDirection = "outbound"
)

type httpConnManagerOptions struct {
	direction         connectionDirection
	rdsRoutConfigName string

	// Additional filters
	wasmStatsHeaders         map[string]string
	extAuthConfig            *auth.ExtAuthConfig
	enableActiveHealthChecks bool

	// Tracing options
	enableTracing      bool
	tracingAPIEndpoint string
}

func (options httpConnManagerOptions) build() (*xds_hcm.HttpConnectionManager, error) {
	connManager := &xds_hcm.HttpConnectionManager{
		StatPrefix: fmt.Sprintf("%s.%s", meshHTTPConnManagerStatPrefix, options.rdsRoutConfigName),
		CodecType:  xds_hcm.HttpConnectionManager_AUTO,
		HttpFilters: []*xds_hcm.HttpFilter{
			// *IMPORTANT NOTE*: The order of filters specified is important.
			// The http_router filter should be the last filter in the chain.
			{
				// HTTP RBAC filter - required to perform HTTP based RBAC per route
				Name: envoy.HTTPRBACFilterName,
				ConfigType: &xds_hcm.HttpFilter_TypedConfig{
					TypedConfig: &any.Any{
						TypeUrl: envoy.HTTPRBACFilterTypeURL,
					},
				},
			},
			{
				Name: envoy.HTTPLocalRateLimitFilterName,
				ConfigType: &xds_hcm.HttpFilter_TypedConfig{
					TypedConfig: protobuf.MustMarshalAny(
						&xds_local_ratelimit.LocalRateLimit{
							StatPrefix: fmt.Sprintf("%s.%s", meshHTTPConnManagerStatPrefix, options.rdsRoutConfigName),
							// Since no token bucket is defined here, the filter is disabled
							// at the listener level. For HTTP traffic, the rate limiting
							// config is applied at the VirtualHost/Route level.
							// Ref: https://www.envoyproxy.io/docs/envoy/latest/configuration/http/http_filters/local_rate_limit_filter#using-rate-limit-descriptors-for-local-rate-limiting
						},
					),
				},
			},
		},
		RouteSpecifier: &xds_hcm.HttpConnectionManager_Rds{
			Rds: &xds_hcm.Rds{
				ConfigSource:    envoy.GetADSConfigSource(),
				RouteConfigName: options.rdsRoutConfigName,
			},
		},
		AccessLog: envoy.GetAccessLog(),
		UpgradeConfigs: []*xds_hcm.HttpConnectionManager_UpgradeConfig{
			{
				UpgradeType: websocketUpgradeType,
			},
		},
	}

	// For inbound connections, add the Authz filter
	if options.direction == inbound && options.extAuthConfig != nil {
		connManager.HttpFilters = append(connManager.HttpFilters, getExtAuthzHTTPFilter(options.extAuthConfig))
	}

	// Enable tracing if requested
	if options.enableTracing {
		tracing, err := getHTTPTracingConfig(options.tracingAPIEndpoint)
		if err != nil {
			return nil, fmt.Errorf("Error getting tracing config for HTTP connection manager: %w", err)
		}

		connManager.GenerateRequestId = &wrappers.BoolValue{
			Value: true,
		}
		connManager.Tracing = tracing
	}

	// Configure WASM stats headers if provided
	if options.wasmStatsHeaders != nil {
		wasmFilters, wasmLocalReplyConfig, err := getWASMStatsConfig(options.wasmStatsHeaders)
		if err != nil {
			return nil, fmt.Errorf("Error getting WASM filters for HTTP connection manager: %w", err)
		}
		connManager.HttpFilters = append(connManager.HttpFilters, wasmFilters...)
		connManager.LocalReplyConfig = wasmLocalReplyConfig
	}

	if options.enableActiveHealthChecks {
		hc, err := getHealthCheckFilter()
		if err != nil {
			return nil, fmt.Errorf("Error getting health check filter for HTTP connection manager: %w", err)
		}
		connManager.HttpFilters = append(connManager.HttpFilters, hc)
	}

	// *IMPORTANT NOTE*: The Router filter must always be the last filter
	connManager.HttpFilters = append(connManager.HttpFilters, &xds_hcm.HttpFilter{
		Name: envoy.HTTPRouterFilterName,
		ConfigType: &xds_hcm.HttpFilter_TypedConfig{
			TypedConfig: &any.Any{
				TypeUrl: envoy.HTTPRouterFilterTypeURL,
			},
		},
	})

	return connManager, nil
}

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
