package bootstrap

import (
	"fmt"
	"time"

	xds_accesslog_filter "github.com/envoyproxy/go-control-plane/envoy/config/accesslog/v3"
	xds_cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	xds_core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	xds_endpoint "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	xds_listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	xds_route "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	xds_accesslog "github.com/envoyproxy/go-control-plane/envoy/extensions/access_loggers/stream/v3"
	xds_http_connection_manager "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	xds_tcp_proxy "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/tcp_proxy/v3"
	"github.com/golang/protobuf/ptypes/any"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/errcode"
	"github.com/openservicemesh/osm/pkg/models"
)

const (
	livenessCluster  = "liveness_cluster"
	readinessCluster = "readiness_cluster"
	startupCluster   = "startup_cluster"

	livenessListener  = "liveness_listener"
	readinessListener = "readiness_listener"
	startupListener   = "startup_listener"
)

func buildProbeCluster(clusterName string, originalProbe *models.HealthProbe) *xds_cluster.Cluster {
	if originalProbe == nil || originalProbe.IsTCPSocket {
		return nil
	}

	return &xds_cluster.Cluster{
		Name: clusterName,
		ClusterDiscoveryType: &xds_cluster.Cluster_Type{
			Type: xds_cluster.Cluster_STATIC,
		},
		LbPolicy: xds_cluster.Cluster_ROUND_ROBIN,
		LoadAssignment: &xds_endpoint.ClusterLoadAssignment{
			ClusterName: clusterName,
			Endpoints: []*xds_endpoint.LocalityLbEndpoints{
				{
					LbEndpoints: []*xds_endpoint.LbEndpoint{
						{
							HostIdentifier: &xds_endpoint.LbEndpoint_Endpoint{
								Endpoint: &xds_endpoint.Endpoint{
									Address: &xds_core.Address{
										Address: &xds_core.Address_SocketAddress{
											SocketAddress: &xds_core.SocketAddress{
												Address: constants.LocalhostIPAddress,
												PortSpecifier: &xds_core.SocketAddress_PortValue{
													PortValue: uint32(originalProbe.Port),
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
		},
	}
}

type probeListenerRoute struct {
	pathPrefixMatch   string
	clusterName       string
	pathPrefixRewrite string
	timeout           time.Duration
}

type probeListenerBuilder struct {
	listenerName      string
	inboundPort       int32
	virtualHostRoutes []probeListenerRoute
	httpsClusterName  string
}

func (plb *probeListenerBuilder) AddProbe(containerName, clusterName, newProbePath string, probe *models.HealthProbe) {
	if probe == nil || probe.IsTCPSocket {
		return
	}

	if probe.IsHTTP {
		plb.virtualHostRoutes = append(plb.virtualHostRoutes, probeListenerRoute{
			pathPrefixMatch:   fmt.Sprintf("%s/%s", newProbePath, containerName),
			clusterName:       clusterName,
			pathPrefixRewrite: probe.Path,
		})
	} else {
		// NOTE: Only 1 HTTPS probe is supported per type (liveness, readiness, startup) across all containers
		// This is due to the fact that we passthrough HTTPS and cannot do any kind of filtering on path
		// Therefore, the last declared HTTPS probe will be the only one receiving traffic
		plb.httpsClusterName = clusterName
	}
}

func (plb *probeListenerBuilder) Build() (*xds_listener.Listener, error) {
	// listenerName should be populated and one of (httpsClusterName, []virtualHostRoutes) should be set
	if plb.listenerName == "" || (plb.httpsClusterName == "" && len(plb.virtualHostRoutes) == 0) {
		return nil, nil
	}

	var filterChains []*xds_listener.FilterChain
	httpAccessLog, err := getHTTPAccessLog()
	if err != nil {
		return nil, err
	}

	// Add the TCPProxy filter first because it has a filter chain match
	if plb.httpsClusterName != "" {
		// NOTE: Only 1 HTTPS probe is supported per type (liveness, readiness, startup)
		// This is due to the fact that we passthrough HTTPS and cannot do any kind of filtering on path
		// Therefore, the last declared HTTPS probe will be the only one receiving traffic
		tcpAccessLog, err := getTCPAccessLog()
		if err != nil {
			return nil, err
		}
		tcpProxy := &xds_tcp_proxy.TcpProxy{
			StatPrefix: "health_probes_https",
			AccessLog: []*xds_accesslog_filter.AccessLog{
				tcpAccessLog,
			},
			ClusterSpecifier: &xds_tcp_proxy.TcpProxy_Cluster{
				Cluster: plb.httpsClusterName,
			},
		}
		pbTCPProxy, err := anypb.New(tcpProxy)
		if err != nil {
			log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrMarshallingXDSResource)).
				Msgf("Error marshaling TcpProxy struct into an anypb.Any message")
			return nil, err
		}
		filterChains = append(filterChains, &xds_listener.FilterChain{
			Filters: []*xds_listener.Filter{
				{
					Name: envoy.TCPProxyFilterName,
					ConfigType: &xds_listener.Filter_TypedConfig{
						TypedConfig: pbTCPProxy,
					},
				},
			},
			// this filter chain match allows for TCPProxy and HTTPConnManager to be used on the same listener
			FilterChainMatch: &xds_listener.FilterChainMatch{
				TransportProtocol: envoy.TransportProtocolTLS,
			},
		})
	}

	if len(plb.virtualHostRoutes) > 0 {
		httpConnectionManager := &xds_http_connection_manager.HttpConnectionManager{
			CodecType:  xds_http_connection_manager.HttpConnectionManager_AUTO,
			StatPrefix: "health_probes_http",
			AccessLog: []*xds_accesslog_filter.AccessLog{
				httpAccessLog,
			},
			RouteSpecifier: &xds_http_connection_manager.HttpConnectionManager_RouteConfig{
				RouteConfig: &xds_route.RouteConfiguration{
					Name: "local_route",
					VirtualHosts: []*xds_route.VirtualHost{
						getVirtualHost(plb.virtualHostRoutes),
					},
				},
			},
			HttpFilters: []*xds_http_connection_manager.HttpFilter{
				{
					Name: envoy.HTTPRouterFilterName,
					ConfigType: &xds_http_connection_manager.HttpFilter_TypedConfig{
						TypedConfig: &any.Any{
							TypeUrl: envoy.HTTPRouterFilterTypeURL,
						},
					},
				},
			},
		}
		pbHTTPConnectionManager, err := anypb.New(httpConnectionManager)
		if err != nil {
			log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrMarshallingXDSResource)).
				Msgf("Error marshaling HttpConnectionManager struct into an anypb.Any message")
			return nil, err
		}
		filterChains = append(filterChains, &xds_listener.FilterChain{
			Filters: []*xds_listener.Filter{
				{
					Name: envoy.HTTPConnectionManagerFilterName,
					ConfigType: &xds_listener.Filter_TypedConfig{
						TypedConfig: pbHTTPConnectionManager,
					},
				},
			},
		})
	}

	return &xds_listener.Listener{
		Name: plb.listenerName,
		Address: &xds_core.Address{
			Address: &xds_core.Address_SocketAddress{
				SocketAddress: &xds_core.SocketAddress{
					Address: "0.0.0.0",
					PortSpecifier: &xds_core.SocketAddress_PortValue{
						PortValue: uint32(plb.inboundPort),
					},
				},
			},
		},
		ListenerFilters: []*xds_listener.ListenerFilter{
			{
				Name: envoy.TLSInspectorFilterName,
				ConfigType: &xds_listener.ListenerFilter_TypedConfig{
					TypedConfig: &anypb.Any{
						TypeUrl: envoy.TLSInspectorFilterTypeURL, // Use TLSInspector to look for HTTPS traffic
					},
				},
			},
		},
		FilterChains: filterChains,
	}, nil
}

func getVirtualHost(routes []probeListenerRoute) *xds_route.VirtualHost {
	var xdsRoutes []*xds_route.Route
	for _, route := range routes {
		routeTimeout := route.timeout
		if routeTimeout < 1*time.Second {
			// This should never happen in practice because the minimum value in Kubernetes
			// is set to 1. However it is easy to check and setting the timeout to 0 will lead
			// to leaks.
			routeTimeout = 1 * time.Second
		}
		xdsRoutes = append(xdsRoutes, &xds_route.Route{
			Match: &xds_route.RouteMatch{
				PathSpecifier: &xds_route.RouteMatch_Prefix{
					Prefix: route.pathPrefixMatch,
				},
			},
			Action: &xds_route.Route_Route{
				Route: &xds_route.RouteAction{
					ClusterSpecifier: &xds_route.RouteAction_Cluster{
						Cluster: route.clusterName,
					},
					PrefixRewrite: route.pathPrefixRewrite,
					Timeout:       durationpb.New(routeTimeout),
				},
			},
		})
	}

	return &xds_route.VirtualHost{
		Name: "local_service",
		Domains: []string{
			"*",
		},
		Routes: xdsRoutes,
	}
}

// getHTTPAccessLog creates an Envoy AccessLog struct.
func getHTTPAccessLog() (*xds_accesslog_filter.AccessLog, error) {
	accessLog, err := anypb.New(getStdoutAccessLog())
	if err != nil {
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrMarshallingXDSResource)).
			Msg("Error marshalling AccessLog object")
		return nil, err
	}
	return &xds_accesslog_filter.AccessLog{
		Name: envoy.AccessLoggerName,
		ConfigType: &xds_accesslog_filter.AccessLog_TypedConfig{
			TypedConfig: accessLog,
		},
	}, nil
}

// getTCPAccessLog creates an Envoy AccessLog struct.
func getTCPAccessLog() (*xds_accesslog_filter.AccessLog, error) {
	accessLog, err := anypb.New(getTCPStdoutAccessLog())
	if err != nil {
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrMarshallingXDSResource)).
			Msg("Error marshalling tcp AccessLog object")
		return nil, err
	}
	return &xds_accesslog_filter.AccessLog{
		Name: envoy.AccessLoggerName,
		ConfigType: &xds_accesslog_filter.AccessLog_TypedConfig{
			TypedConfig: accessLog,
		},
	}, nil
}

func getStdoutAccessLog() *xds_accesslog.StdoutAccessLog {
	accessLogger := &xds_accesslog.StdoutAccessLog{
		AccessLogFormat: &xds_accesslog.StdoutAccessLog_LogFormat{
			LogFormat: &xds_core.SubstitutionFormatString{
				Format: &xds_core.SubstitutionFormatString_JsonFormat{
					JsonFormat: &structpb.Struct{
						Fields: map[string]*structpb.Value{
							"start_time":            pbStringValue(`%START_TIME%`),
							"method":                pbStringValue(`%REQ(:METHOD)%`),
							"path":                  pbStringValue(`%REQ(X-ENVOY-ORIGINAL-PATH?:PATH)%`),
							"protocol":              pbStringValue(`%PROTOCOL%`),
							"response_code":         pbStringValue(`%RESPONSE_CODE%`),
							"response_code_details": pbStringValue(`%RESPONSE_CODE_DETAILS%`),
							"time_to_first_byte":    pbStringValue(`%RESPONSE_DURATION%`),
							"upstream_cluster":      pbStringValue(`%UPSTREAM_CLUSTER%`),
							"response_flags":        pbStringValue(`%RESPONSE_FLAGS%`),
							"bytes_received":        pbStringValue(`%BYTES_RECEIVED%`),
							"bytes_sent":            pbStringValue(`%BYTES_SENT%`),
							"duration":              pbStringValue(`%DURATION%`),
							"upstream_service_time": pbStringValue(`%RESP(X-ENVOY-UPSTREAM-SERVICE-TIME)%`),
							"x_forwarded_for":       pbStringValue(`%REQ(X-FORWARDED-FOR)%`),
							"user_agent":            pbStringValue(`%REQ(USER-AGENT)%`),
							"request_id":            pbStringValue(`%REQ(X-REQUEST-ID)%`),
							"requested_server_name": pbStringValue("%REQUESTED_SERVER_NAME%"),
							"authority":             pbStringValue(`%REQ(:AUTHORITY)%`),
							"upstream_host":         pbStringValue(`%UPSTREAM_HOST%`),
						},
					},
				},
			},
		},
	}
	return accessLogger
}

func getTCPStdoutAccessLog() *xds_accesslog.StdoutAccessLog {
	accessLogger := &xds_accesslog.StdoutAccessLog{
		AccessLogFormat: &xds_accesslog.StdoutAccessLog_LogFormat{
			LogFormat: &xds_core.SubstitutionFormatString{
				Format: &xds_core.SubstitutionFormatString_JsonFormat{
					JsonFormat: &structpb.Struct{
						Fields: map[string]*structpb.Value{
							"start_time":            pbStringValue(`%START_TIME%`),
							"upstream_cluster":      pbStringValue(`%UPSTREAM_CLUSTER%`),
							"response_flags":        pbStringValue(`%RESPONSE_FLAGS%`),
							"bytes_received":        pbStringValue(`%BYTES_RECEIVED%`),
							"bytes_sent":            pbStringValue(`%BYTES_SENT%`),
							"duration":              pbStringValue(`%DURATION%`),
							"requested_server_name": pbStringValue("%REQUESTED_SERVER_NAME%"),
							"upstream_host":         pbStringValue(`%UPSTREAM_HOST%`),
						},
					},
				},
			},
		},
	}
	return accessLogger
}

func pbStringValue(v string) *structpb.Value {
	return &structpb.Value{
		Kind: &structpb.Value_StringValue{
			StringValue: v,
		},
	}
}
