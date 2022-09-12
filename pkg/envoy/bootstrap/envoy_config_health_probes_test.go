package bootstrap

import (
	"fmt"
	"testing"
	"time"

	xds_accesslog_filter "github.com/envoyproxy/go-control-plane/envoy/config/accesslog/v3"
	xds_cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	xds_core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	xds_listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	xds_route "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	xds_http_connection_manager "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	xds_tcp_proxy "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/tcp_proxy/v3"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/onsi/ginkgo"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/envoy/bootstrap/test"
	"github.com/openservicemesh/osm/pkg/models"
)

var _ = ginkgo.Describe("Test functions creating Envoy config and rewriting the Pod's health probes to pass through Envoy", func() {

	timeout := 42 * time.Second
	liveness := &models.HealthProbe{Path: "/liveness", Port: 81, IsHTTP: true, IsTCPSocket: false, Timeout: timeout}
	readiness := &models.HealthProbe{Path: "/readiness", Port: 82, IsHTTP: true, IsTCPSocket: false, Timeout: timeout}
	startup := &models.HealthProbe{Path: "/startup", Port: 83, IsHTTP: true, IsTCPSocket: false, Timeout: timeout}

	var routes []probeListenerRoute
	var defaultTimeoutRoutes []probeListenerRoute
	routes = append(routes, probeListenerRoute{
		pathPrefixRewrite: "/original/probe/path",
		clusterName:       "-cluster-name-",
		pathPrefixMatch:   "/some/path",
		timeout:           timeout,
	})

	defaultTimeoutRoutes = append(routes, probeListenerRoute{
		pathPrefixRewrite: "/original/probe/path",
		clusterName:       "-cluster-name-2-",
		pathPrefixMatch:   "/some/path2",
		timeout:           0 * time.Second,
	})

	var allRoutes []probeListenerRoute
	allRoutes = append(allRoutes, routes...)
	allRoutes = append(allRoutes, defaultTimeoutRoutes...)
	// Listed below are the functions we are going to test.
	// The key in the map is the name of the function -- must match what's in the value of the map.
	// The key (function name) is used to locate and load the YAML file with the expected return for this function.
	clusterFunctionsToTest := map[string]func() protoreflect.ProtoMessage{
		"getVirtualHosts": func() protoreflect.ProtoMessage {
			return getVirtualHost(routes)
		},
		"getVirtualHostsDefault": func() protoreflect.ProtoMessage {
			return getVirtualHost(defaultTimeoutRoutes)
		},
		"getVirtualHostsMultiple": func() protoreflect.ProtoMessage {
			return getVirtualHost(allRoutes)
		},
		"getProbeCluster":     func() protoreflect.ProtoMessage { return getProbeCluster("cluster-name", 12341234) },
		"getLivenessCluster":  func() protoreflect.ProtoMessage { return getLivenessCluster("my-container", liveness) },
		"getReadinessCluster": func() protoreflect.ProtoMessage { return getReadinessCluster("my-container", readiness) },
		"getStartupCluster":   func() protoreflect.ProtoMessage { return getStartupCluster("my-container", startup) },
	}

	listenerFunctionsToTest := map[string]func() (protoreflect.ProtoMessage, error){
		"getHTTPAccessLog": func() (protoreflect.ProtoMessage, error) { return getHTTPAccessLog() },
		"getTCPAccessLog":  func() (protoreflect.ProtoMessage, error) { return getTCPAccessLog() },
	}

	for fnName, fn := range clusterFunctionsToTest {
		// A call to test.ThisFunction will:
		//     a) marshal return xDS struct of each function to yaml (and save it to "actual_output_<functionName>.yaml")
		//     b) load expectation from "expected_output_<functionName>.yaml"
		//     c) compare actual and expected in a ginkgo.Context() + ginkgo.It()
		test.ThisXdsClusterFunction(fnName, fn)
	}

	for fnName, fn := range listenerFunctionsToTest {
		// A call to test.ThisFunction will:
		//     a) check for error
		//     b) marshal return xDS struct of each function to yaml (and save it to "actual_output_<functionName>.yaml")
		//     c) load expectation from "expected_output_<functionName>.yaml"
		//     d) compare actual and expected in a ginkgo.Context() + ginkgo.It()
		test.ThisXdsListenerFunction(fnName, fn)
	}
})

func TestGetProbeCluster(t *testing.T) {
	type probeClusterTest struct {
		conainerName string
		name         string
		probe        *models.HealthProbe
		expected     *xds_cluster.Cluster
	}

	t.Run("liveness", func(t *testing.T) {
		tests := []probeClusterTest{
			{
				name: "nil",
			},
		}
		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				assert.Equal(t, test.expected, getLivenessCluster(test.conainerName, test.probe))
			})
		}
	})

	t.Run("readiness", func(t *testing.T) {
		tests := []probeClusterTest{
			{
				name: "nil",
			},
		}
		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				assert.Equal(t, test.expected, getReadinessCluster(test.conainerName, test.probe))
			})
		}
	})

	t.Run("startup", func(t *testing.T) {
		tests := []probeClusterTest{
			{
				name: "nil",
			},
		}
		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				assert.Equal(t, test.expected, getStartupCluster(test.conainerName, test.probe))
			})
		}
	})
}

func Test_probeListenerBuilder_Build(t *testing.T) {
	timeout := 42 * time.Second
	liveness := &models.HealthProbe{Path: "/liveness", Port: 81, IsHTTP: true, IsTCPSocket: false, Timeout: timeout}
	readiness := &models.HealthProbe{Path: "/readiness", Port: 82, IsHTTP: true, IsTCPSocket: false, Timeout: timeout}
	httpAccessLog, err := getHTTPAccessLog()
	if err != nil {
		t.Fatal(err)
	}

	tcpAccessLog, err := getTCPAccessLog()
	if err != nil {
		t.Fatal(err)
	}

	testLivenessListenerHTTPConnManager := &xds_http_connection_manager.HttpConnectionManager{
		CodecType:  xds_http_connection_manager.HttpConnectionManager_AUTO,
		StatPrefix: "health_probes_http",
		AccessLog: []*xds_accesslog_filter.AccessLog{
			httpAccessLog,
		},
		RouteSpecifier: &xds_http_connection_manager.HttpConnectionManager_RouteConfig{
			RouteConfig: &xds_route.RouteConfiguration{
				Name: "local_route",
				VirtualHosts: []*xds_route.VirtualHost{
					{
						Name: "local_service",
						Domains: []string{
							"*",
						},
						Routes: []*xds_route.Route{
							{
								Match: &xds_route.RouteMatch{
									PathSpecifier: &xds_route.RouteMatch_Prefix{
										Prefix: "/osm-liveness-probe/my-container",
									},
								},
								Action: &xds_route.Route_Route{
									Route: &xds_route.RouteAction{
										ClusterSpecifier: &xds_route.RouteAction_Cluster{
											Cluster: "my-container_liveness_cluster",
										},
										PrefixRewrite: "/liveness",
										Timeout:       durationpb.New(timeout),
									},
								},
							},
						},
					},
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

	pbHTTPConnectionManager, err := anypb.New(testLivenessListenerHTTPConnManager)
	if err != nil {
		t.Fatal(err)
	}
	testLivenessListener := &xds_listener.Listener{
		Name: "liveness_listener",
		Address: &xds_core.Address{
			Address: &xds_core.Address_SocketAddress{
				SocketAddress: &xds_core.SocketAddress{
					Address: "0.0.0.0",
					PortSpecifier: &xds_core.SocketAddress_PortValue{
						PortValue: 15901,
					},
				},
			},
		},
		FilterChains: []*xds_listener.FilterChain{
			{
				Filters: []*xds_listener.Filter{
					{
						Name: envoy.HTTPConnectionManagerFilterName,
						ConfigType: &xds_listener.Filter_TypedConfig{
							TypedConfig: pbHTTPConnectionManager,
						},
					},
				},
			},
		},
	}

	testReadinessListenerHTTPConnManager := &xds_http_connection_manager.HttpConnectionManager{
		CodecType:  xds_http_connection_manager.HttpConnectionManager_AUTO,
		StatPrefix: "health_probes_http",
		AccessLog: []*xds_accesslog_filter.AccessLog{
			httpAccessLog,
		},
		RouteSpecifier: &xds_http_connection_manager.HttpConnectionManager_RouteConfig{
			RouteConfig: &xds_route.RouteConfiguration{
				Name: "local_route",
				VirtualHosts: []*xds_route.VirtualHost{
					{
						Name: "local_service",
						Domains: []string{
							"*",
						},
						Routes: []*xds_route.Route{
							{
								Match: &xds_route.RouteMatch{
									PathSpecifier: &xds_route.RouteMatch_Prefix{
										Prefix: "/osm-readiness-probe/my-container",
									},
								},
								Action: &xds_route.Route_Route{
									Route: &xds_route.RouteAction{
										ClusterSpecifier: &xds_route.RouteAction_Cluster{
											Cluster: "my-container_readiness_cluster",
										},
										PrefixRewrite: "/readiness",
										Timeout:       durationpb.New(timeout),
									},
								},
							},
							{
								Match: &xds_route.RouteMatch{
									PathSpecifier: &xds_route.RouteMatch_Prefix{
										Prefix: "/osm-readiness-probe/my-sidecar",
									},
								},
								Action: &xds_route.Route_Route{
									Route: &xds_route.RouteAction{
										ClusterSpecifier: &xds_route.RouteAction_Cluster{
											Cluster: "my-sidecar_readiness_cluster",
										},
										PrefixRewrite: "/readiness",
										Timeout:       durationpb.New(1 * time.Second),
									},
								},
							},
						},
					},
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

	pbHTTPConnectionManagerReadiness, err := anypb.New(testReadinessListenerHTTPConnManager)
	if err != nil {
		t.Fatal(err)
	}
	testReadinessListener := &xds_listener.Listener{
		Name: "readiness_listener",
		Address: &xds_core.Address{
			Address: &xds_core.Address_SocketAddress{
				SocketAddress: &xds_core.SocketAddress{
					Address: "0.0.0.0",
					PortSpecifier: &xds_core.SocketAddress_PortValue{
						PortValue: 15902,
					},
				},
			},
		},
		FilterChains: []*xds_listener.FilterChain{
			{
				Filters: []*xds_listener.Filter{
					{
						Name: envoy.HTTPConnectionManagerFilterName,
						ConfigType: &xds_listener.Filter_TypedConfig{
							TypedConfig: pbHTTPConnectionManagerReadiness,
						},
					},
				},
			},
		},
	}

	httpsProbeTCPProxy := &xds_tcp_proxy.TcpProxy{
		StatPrefix: "health_probes",
		AccessLog: []*xds_accesslog_filter.AccessLog{
			tcpAccessLog,
		},
		ClusterSpecifier: &xds_tcp_proxy.TcpProxy_Cluster{
			Cluster: "my-sidecar_startup_cluster",
		},
	}
	pbTCPProxy, err := anypb.New(httpsProbeTCPProxy)
	if err != nil {
		t.Fatal(err)
	}

	httpsListener := &xds_listener.Listener{
		Name: "startup_listener",
		Address: &xds_core.Address{
			Address: &xds_core.Address_SocketAddress{
				SocketAddress: &xds_core.SocketAddress{
					Address: "0.0.0.0",
					PortSpecifier: &xds_core.SocketAddress_PortValue{
						PortValue: 15903,
					},
				},
			},
		},
		FilterChains: []*xds_listener.FilterChain{
			{
				Filters: []*xds_listener.Filter{
					{
						Name: envoy.TCPProxyFilterName,
						ConfigType: &xds_listener.Filter_TypedConfig{
							TypedConfig: pbTCPProxy,
						},
					},
				},
			},
		},
	}

	type fields struct {
		listenerName      string
		inboundPort       int32
		virtualHostRoutes []probeListenerRoute
		isHTTP            bool
		httpsClusterName  string
	}
	tests := []struct {
		name    string
		fields  fields
		want    *xds_listener.Listener
		wantErr bool
	}{
		{
			name: "livenessListener",
			fields: fields{
				listenerName: livenessListener,
				inboundPort:  constants.LivenessProbePort,
				isHTTP:       true,
				virtualHostRoutes: []probeListenerRoute{
					{
						clusterName:       fmt.Sprintf("my-container_%s", livenessCluster),
						pathPrefixMatch:   fmt.Sprintf("%s/my-container", constants.LivenessProbePath),
						pathPrefixRewrite: liveness.Path,
						timeout:           timeout,
					},
				},
			},
			want: testLivenessListener,
		},
		{
			name: "readinessListener (multiple routes, default timeout)",
			fields: fields{
				listenerName: readinessListener,
				inboundPort:  constants.ReadinessProbePort,
				isHTTP:       true,
				virtualHostRoutes: []probeListenerRoute{
					{
						clusterName:       fmt.Sprintf("my-container_%s", readinessCluster),
						pathPrefixMatch:   fmt.Sprintf("%s/my-container", constants.ReadinessProbePath),
						pathPrefixRewrite: readiness.Path,
						timeout:           timeout,
					},
					{
						clusterName:       fmt.Sprintf("my-sidecar_%s", readinessCluster),
						pathPrefixMatch:   fmt.Sprintf("%s/my-sidecar", constants.ReadinessProbePath),
						pathPrefixRewrite: readiness.Path,
						timeout:           0 * time.Second, // Build() should change 0s to 1s
					},
				},
			},
			want: testReadinessListener,
		},

		{
			name: "HTTPS probe",
			fields: fields{
				listenerName:     "startup_listener",
				httpsClusterName: fmt.Sprintf("my-sidecar_%s", startupCluster),
				inboundPort:      constants.StartupProbePort,
			},
			want: httpsListener,
		},
		{
			name: "http: no virtualHosts (should return nil and no error)",
			fields: fields{
				listenerName: "my-listener",
				isHTTP:       true,
			},
		},
		{
			name: "no listener name (should return nil and no error)",
			fields: fields{
				virtualHostRoutes: []probeListenerRoute{
					{},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plb := &probeListenerBuilder{
				listenerName:      tt.fields.listenerName,
				inboundPort:       tt.fields.inboundPort,
				virtualHostRoutes: tt.fields.virtualHostRoutes,
				isHTTP:            tt.fields.isHTTP,
				httpsClusterName:  tt.fields.httpsClusterName,
			}
			marshalOptions := protojson.MarshalOptions{
				UseProtoNames: true,
			}
			got, err := plb.Build()
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)

			wantJSON, err := marshalOptions.Marshal(tt.want)
			if err != nil {
				t.Fatalf("error encountered marshalling tt.want to JSON: %s", err)
			}

			gotJSON, err := marshalOptions.Marshal(got)
			if err != nil {
				t.Fatalf("error encountered marshalling plb.Build() to JSON: %s", err)
			}

			assert.Equal(t, string(wantJSON), string(gotJSON))
		})
	}
}
