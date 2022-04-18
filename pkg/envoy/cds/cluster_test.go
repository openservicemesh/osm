package cds

import (
	"math"
	"testing"
	"time"

	xds_cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	xds_core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	xds_endpoint "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	"github.com/golang/mock/gomock"
	"github.com/golang/protobuf/ptypes/wrappers"
	tassert "github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/wrapperspb"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	configv1alpha3 "github.com/openservicemesh/osm/pkg/apis/config/v1alpha3"
	policyv1alpha1 "github.com/openservicemesh/osm/pkg/apis/policy/v1alpha1"

	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/tests"
	"github.com/openservicemesh/osm/pkg/trafficpolicy"
)

func TestGetUpstreamServiceCluster(t *testing.T) {
	var thresholdUintVal uint32 = 3
	thresholdDuration := &metav1.Duration{Duration: time.Duration(1 * time.Second)}

	downstreamSvcAccount := tests.BookbuyerServiceIdentity
	upstreamSvc := service.MeshService{
		Namespace: "default",
		Name:      "bookstore-v1",
		Port:      14001,
	}
	testCases := []struct {
		name                            string
		clusterConfig                   trafficpolicy.MeshClusterConfig
		expectedCircuitBreakerThreshold *xds_cluster.CircuitBreakers
	}{
		{
			name: "EDS based cluster adds health checks when configured",
			clusterConfig: trafficpolicy.MeshClusterConfig{
				Name:                          "default/bookstore-v1_14001",
				Service:                       upstreamSvc,
				EnableEnvoyActiveHealthChecks: true,
			},
		},
		{
			name: "EDS based cluster does not add health checks when not configured",
			clusterConfig: trafficpolicy.MeshClusterConfig{
				Name:                          "default/bookstore-v1_14001",
				Service:                       upstreamSvc,
				EnableEnvoyActiveHealthChecks: false,
			},
		},
		{
			name: "Cluster with Circuit Breaker",
			clusterConfig: trafficpolicy.MeshClusterConfig{
				Name:    "default/bookstore-v1_14001",
				Service: upstreamSvc,
				UpstreamTrafficSetting: &policyv1alpha1.UpstreamTrafficSetting{
					Spec: policyv1alpha1.UpstreamTrafficSettingSpec{
						ConnectionSettings: &policyv1alpha1.ConnectionSettingsSpec{
							TCP: &policyv1alpha1.TCPConnectionSettings{
								MaxConnections: &thresholdUintVal,
								ConnectTimeout: thresholdDuration,
							},
							HTTP: &policyv1alpha1.HTTPConnectionSettings{
								MaxRequests:              &thresholdUintVal,
								MaxPendingRequests:       &thresholdUintVal,
								MaxRetries:               &thresholdUintVal,
								MaxRequestsPerConnection: &thresholdUintVal,
							},
						},
					},
				},
			},
			expectedCircuitBreakerThreshold: &xds_cluster.CircuitBreakers{
				Thresholds: []*xds_cluster.CircuitBreakers_Thresholds{
					{
						MaxConnections:     wrapperspb.UInt32(thresholdUintVal),
						MaxRequests:        wrapperspb.UInt32(thresholdUintVal),
						MaxPendingRequests: wrapperspb.UInt32(thresholdUintVal),
						MaxRetries:         wrapperspb.UInt32(thresholdUintVal),
						TrackRemaining:     true,
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := tassert.New(t)

			remoteCluster := getUpstreamServiceCluster(downstreamSvcAccount, tc.clusterConfig, configv1alpha3.SidecarSpec{})
			assert.NotNil(remoteCluster)

			if tc.clusterConfig.EnableEnvoyActiveHealthChecks {
				assert.NotNil(remoteCluster.HealthChecks)
			} else {
				assert.Nil(remoteCluster.HealthChecks)
			}

			if tc.clusterConfig.UpstreamTrafficSetting != nil {
				assert.Equal(tc.expectedCircuitBreakerThreshold, remoteCluster.CircuitBreakers)
			}
		})
	}
}

func TestGetMulticlusterGatewayUpstreamServiceCluster(t *testing.T) {
	upstreamSvc := service.MeshService{
		Namespace:  "ns1",
		Name:       "s1",
		TargetPort: 8080,
	}

	testCases := []struct {
		name                        string
		expectedClusterType         xds_cluster.Cluster_DiscoveryType
		expectedLbPolicy            xds_cluster.Cluster_LbPolicy
		expectedLocalityLbEndpoints []*xds_endpoint.LocalityLbEndpoints
		addHealthCheck              bool
	}{
		{
			name:                "Gateway upstream cluster configuration with health checks configured",
			expectedClusterType: xds_cluster.Cluster_STRICT_DNS,
			expectedLbPolicy:    xds_cluster.Cluster_ROUND_ROBIN,
			expectedLocalityLbEndpoints: []*xds_endpoint.LocalityLbEndpoints{
				{
					LbEndpoints: []*xds_endpoint.LbEndpoint{{
						HostIdentifier: &xds_endpoint.LbEndpoint_Endpoint{
							Endpoint: &xds_endpoint.Endpoint{
								Address: envoy.GetAddress("s1.ns1.svc.cluster.local", uint32(8080)),
							},
						},
					}},
				},
			},
			addHealthCheck: true,
		},
		{
			name:                "Gateway upstream cluster configuration with health checks not configured",
			expectedClusterType: xds_cluster.Cluster_STRICT_DNS,
			expectedLbPolicy:    xds_cluster.Cluster_ROUND_ROBIN,
			expectedLocalityLbEndpoints: []*xds_endpoint.LocalityLbEndpoints{
				{
					LbEndpoints: []*xds_endpoint.LbEndpoint{{
						HostIdentifier: &xds_endpoint.LbEndpoint_Endpoint{
							Endpoint: &xds_endpoint.Endpoint{
								Address: envoy.GetAddress("s1.ns1.svc.cluster.local", uint32(8080)),
							},
						},
					}},
				},
			},
			addHealthCheck: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := tassert.New(t)
			mockCtrl := gomock.NewController(t)
			mockCatalog := catalog.NewMockMeshCataloger(mockCtrl)

			remoteCluster, err := getMulticlusterGatewayUpstreamServiceCluster(mockCatalog, upstreamSvc, tc.addHealthCheck)
			assert.NoError(err)
			assert.Equal(tc.expectedClusterType, remoteCluster.GetType())
			assert.Equal(tc.expectedLbPolicy, remoteCluster.LbPolicy)
			assert.Equal(len(tc.expectedLocalityLbEndpoints), len(remoteCluster.LoadAssignment.Endpoints))
			assert.ElementsMatch(tc.expectedLocalityLbEndpoints, remoteCluster.LoadAssignment.Endpoints)
			assert.Equal(remoteCluster.LoadAssignment.ClusterName, upstreamSvc.ServerName())

			if tc.addHealthCheck {
				assert.NotNil(remoteCluster.HealthChecks)
			} else {
				assert.Nil(remoteCluster.HealthChecks)
			}
		})
	}
}

func TestGetLocalServiceCluster(t *testing.T) {
	testCases := []struct {
		name                             string
		clusterConfig                    trafficpolicy.MeshClusterConfig
		expectedLocalityLbEndpoints      []*xds_endpoint.LocalityLbEndpoints
		expectedLbPolicy                 xds_cluster.Cluster_LbPolicy
		expectedProtocolSelection        xds_cluster.Cluster_ClusterProtocolSelection
		expectedPortToProtocolMappingErr bool
		expectedErr                      bool
	}{
		{
			name: "Local service cluster",
			clusterConfig: trafficpolicy.MeshClusterConfig{
				Name:    "ns/foo|90|local",
				Service: service.MeshService{Namespace: "ns", Name: "foo"},
				Port:    90,
				Address: "127.0.0.1",
			},
			expectedLocalityLbEndpoints: []*xds_endpoint.LocalityLbEndpoints{
				{
					Locality: &xds_core.Locality{
						Zone: "zone",
					},
					LbEndpoints: []*xds_endpoint.LbEndpoint{{
						HostIdentifier: &xds_endpoint.LbEndpoint_Endpoint{
							Endpoint: &xds_endpoint.Endpoint{
								Address: envoy.GetAddress("127.0.0.1", uint32(90)),
							},
						},
						LoadBalancingWeight: &wrappers.UInt32Value{
							Value: constants.ClusterWeightAcceptAll, // Local cluster accepts all traffic
						},
					}},
				},
			},
			expectedErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := tassert.New(t)

			cluster := getLocalServiceCluster(tc.clusterConfig)

			if tc.expectedErr {
				assert.Nil(cluster)
			} else {
				assert.Equal(tc.clusterConfig.Name, cluster.Name)
				assert.Equal(tc.clusterConfig.Name, cluster.AltStatName)
				assert.Equal(xds_cluster.Cluster_ROUND_ROBIN, cluster.LbPolicy)
				assert.Equal(&xds_cluster.Cluster_Type{Type: xds_cluster.Cluster_STRICT_DNS}, cluster.ClusterDiscoveryType)
				assert.Equal(true, cluster.RespectDnsTtl)
				assert.Equal(xds_cluster.Cluster_V4_ONLY, cluster.DnsLookupFamily)
				assert.Equal(len(tc.expectedLocalityLbEndpoints), len(cluster.LoadAssignment.Endpoints))
				assert.ElementsMatch(tc.expectedLocalityLbEndpoints, cluster.LoadAssignment.Endpoints)
			}
		})
	}
}

func TestGetPrometheusCluster(t *testing.T) {
	assert := tassert.New(t)

	expectedCluster := &xds_cluster.Cluster{
		TransportSocketMatches: nil,
		Name:                   constants.EnvoyMetricsCluster,
		AltStatName:            constants.EnvoyMetricsCluster,
		ClusterDiscoveryType:   &xds_cluster.Cluster_Type{Type: xds_cluster.Cluster_STATIC},
		EdsClusterConfig:       nil,
		LoadAssignment: &xds_endpoint.ClusterLoadAssignment{
			ClusterName: constants.EnvoyMetricsCluster,
			Endpoints: []*xds_endpoint.LocalityLbEndpoints{
				{
					Locality: nil,
					LbEndpoints: []*xds_endpoint.LbEndpoint{{
						HostIdentifier: &xds_endpoint.LbEndpoint_Endpoint{
							Endpoint: &xds_endpoint.Endpoint{
								Address: &xds_core.Address{
									Address: &xds_core.Address_SocketAddress{
										SocketAddress: &xds_core.SocketAddress{
											Protocol: xds_core.SocketAddress_TCP,
											Address:  "127.0.0.1",
											PortSpecifier: &xds_core.SocketAddress_PortValue{
												PortValue: uint32(15000),
											},
										},
									},
								},
							},
						},
						LoadBalancingWeight: &wrappers.UInt32Value{
							Value: 100,
						},
					}},
				},
			},
		},
	}

	actual := *getPrometheusCluster()
	assert.Equal(expectedCluster.LoadAssignment.ClusterName, actual.LoadAssignment.ClusterName)
	assert.Equal(len(expectedCluster.LoadAssignment.Endpoints[0].LbEndpoints), len(actual.LoadAssignment.Endpoints))
	assert.Equal(expectedCluster.LoadAssignment.Endpoints[0].LbEndpoints, actual.LoadAssignment.Endpoints[0].LbEndpoints)
	assert.Equal(expectedCluster.LoadAssignment, actual.LoadAssignment)
	assert.Equal(expectedCluster, &actual)
}

func TestGetOriginalDestinationEgressCluster(t *testing.T) {
	assert := tassert.New(t)
	typedHTTPProtocolOptions, err := getTypedHTTPProtocolOptions(getDefaultHTTPProtocolOptions())
	assert.Nil(err)

	var thresholdUintVal uint32 = 3
	thresholdDuration := &metav1.Duration{Duration: time.Duration(1 * time.Second)}

	testCases := []struct {
		name                   string
		clusterName            string
		upstreamTrafficSetting *policyv1alpha1.UpstreamTrafficSetting
		expected               *xds_cluster.Cluster
	}{
		{
			name:        "foo cluster without UpstreamTrafficSetting specified",
			clusterName: "foo",
			expected: &xds_cluster.Cluster{
				Name: "foo",
				ClusterDiscoveryType: &xds_cluster.Cluster_Type{
					Type: xds_cluster.Cluster_ORIGINAL_DST,
				},
				LbPolicy:                      xds_cluster.Cluster_CLUSTER_PROVIDED,
				TypedExtensionProtocolOptions: typedHTTPProtocolOptions,
				CircuitBreakers: &xds_cluster.CircuitBreakers{
					Thresholds: []*xds_cluster.CircuitBreakers_Thresholds{
						{
							MaxConnections:     wrapperspb.UInt32(math.MaxUint32),
							MaxRequests:        wrapperspb.UInt32(math.MaxUint32),
							MaxPendingRequests: wrapperspb.UInt32(math.MaxUint32),
							MaxRetries:         wrapperspb.UInt32(math.MaxUint32),
							TrackRemaining:     true,
						},
					},
				},
			},
		},
		{
			name:        "bar cluster with UpstreamTrafficSetting specified",
			clusterName: "bar",
			upstreamTrafficSetting: &policyv1alpha1.UpstreamTrafficSetting{
				Spec: policyv1alpha1.UpstreamTrafficSettingSpec{
					ConnectionSettings: &policyv1alpha1.ConnectionSettingsSpec{
						TCP: &policyv1alpha1.TCPConnectionSettings{
							MaxConnections: &thresholdUintVal,
							ConnectTimeout: thresholdDuration,
						},
						HTTP: &policyv1alpha1.HTTPConnectionSettings{
							MaxRequests:              &thresholdUintVal,
							MaxPendingRequests:       &thresholdUintVal,
							MaxRetries:               &thresholdUintVal,
							MaxRequestsPerConnection: &thresholdUintVal,
						},
					},
				},
			},
			expected: &xds_cluster.Cluster{
				Name: "bar",
				ClusterDiscoveryType: &xds_cluster.Cluster_Type{
					Type: xds_cluster.Cluster_ORIGINAL_DST,
				},
				LbPolicy:                      xds_cluster.Cluster_CLUSTER_PROVIDED,
				TypedExtensionProtocolOptions: typedHTTPProtocolOptions,
				ConnectTimeout:                durationpb.New(thresholdDuration.Duration),
				MaxRequestsPerConnection:      wrapperspb.UInt32(thresholdUintVal),
				CircuitBreakers: &xds_cluster.CircuitBreakers{
					Thresholds: []*xds_cluster.CircuitBreakers_Thresholds{
						{
							MaxConnections:     wrapperspb.UInt32(thresholdUintVal),
							MaxRequests:        wrapperspb.UInt32(thresholdUintVal),
							MaxPendingRequests: wrapperspb.UInt32(thresholdUintVal),
							MaxRetries:         wrapperspb.UInt32(thresholdUintVal),
							TrackRemaining:     true,
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := tassert.New(t)

			actual, err := getOriginalDestinationEgressCluster(tc.clusterName, tc.upstreamTrafficSetting)
			assert.Nil(err)
			assert.Equal(tc.expected, actual)
		})
	}
}

func TestGetEgressClusters(t *testing.T) {
	testCases := []struct {
		name                 string
		clusterConfigs       []*trafficpolicy.EgressClusterConfig
		expectedClusterCount int
	}{
		{
			name:                 "no cluster configs specified",
			clusterConfigs:       nil,
			expectedClusterCount: 0,
		},
		{
			name: "all cluster configs are valid HTTP clusters",
			clusterConfigs: []*trafficpolicy.EgressClusterConfig{
				{
					Name: "foo.com:80",
					Host: "foo.com",
					Port: 80,
				},
				{
					Name: "bar.com:90",
					Host: "bar.com",
					Port: 90,
				},
			},
			expectedClusterCount: 2,
		},
		{
			name: "some cluster configs are invalid HTTP clusters",
			clusterConfigs: []*trafficpolicy.EgressClusterConfig{
				{
					Name: "foo.com:80",
					Host: "foo.com",
					Port: 80,
				},
				{
					Name: "bar.com:90",
					Port: 90,
				},
			},
			expectedClusterCount: 2,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := tassert.New(t)

			actual := getEgressClusters(tc.clusterConfigs)
			assert.Len(actual, tc.expectedClusterCount)
		})
	}
}

func TestGetDNSResolvableEgressCluster(t *testing.T) {
	typedHTTPProtocolOptions, _ := getTypedHTTPProtocolOptions(getDefaultHTTPProtocolOptions())

	testCases := []struct {
		name            string
		clusterConfig   *trafficpolicy.EgressClusterConfig
		expectedCluster *xds_cluster.Cluster
		expectError     bool
	}{
		{
			name:            "egress cluster config is nil",
			clusterConfig:   nil,
			expectedCluster: nil,
			expectError:     true,
		},
		{
			name: "valid egress cluster config",
			clusterConfig: &trafficpolicy.EgressClusterConfig{
				Name: "foo.com:80",
				Host: "foo.com",
				Port: 80,
			},
			expectedCluster: &xds_cluster.Cluster{
				Name:        "foo.com:80",
				AltStatName: "foo_com_80",
				ClusterDiscoveryType: &xds_cluster.Cluster_Type{
					Type: xds_cluster.Cluster_STRICT_DNS,
				},
				LbPolicy: xds_cluster.Cluster_ROUND_ROBIN,
				LoadAssignment: &xds_endpoint.ClusterLoadAssignment{
					ClusterName: "foo.com:80",
					Endpoints: []*xds_endpoint.LocalityLbEndpoints{
						{
							LbEndpoints: []*xds_endpoint.LbEndpoint{{
								HostIdentifier: &xds_endpoint.LbEndpoint_Endpoint{
									Endpoint: &xds_endpoint.Endpoint{
										Address: envoy.GetAddress("foo.com", 80),
									},
								},
								LoadBalancingWeight: &wrappers.UInt32Value{
									Value: constants.ClusterWeightAcceptAll,
								},
							}},
						},
					},
				},
				TypedExtensionProtocolOptions: typedHTTPProtocolOptions,
				CircuitBreakers: &xds_cluster.CircuitBreakers{
					Thresholds: []*xds_cluster.CircuitBreakers_Thresholds{
						{
							MaxConnections:     wrapperspb.UInt32(math.MaxUint32),
							MaxRequests:        wrapperspb.UInt32(math.MaxUint32),
							MaxPendingRequests: wrapperspb.UInt32(math.MaxUint32),
							MaxRetries:         wrapperspb.UInt32(math.MaxUint32),
							TrackRemaining:     true,
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "egress cluster config Name unspecified",
			clusterConfig: &trafficpolicy.EgressClusterConfig{
				Host: "foo.com",
				Port: 80,
			},
			expectedCluster: nil,
			expectError:     true,
		},
		{
			name: "egress cluster config Host unspecified",
			clusterConfig: &trafficpolicy.EgressClusterConfig{
				Name: "foo.com:80",
				Port: 80,
			},
			expectedCluster: nil,
			expectError:     true,
		},
		{
			name: "egress cluster config Port unspecified",
			clusterConfig: &trafficpolicy.EgressClusterConfig{
				Name: "foo.com:80",
				Host: "foo.com",
			},
			expectedCluster: nil,
			expectError:     true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := tassert.New(t)

			actual, err := getDNSResolvableEgressCluster(tc.clusterConfig)
			assert.Equal(tc.expectError, err != nil)
			assert.Equal(tc.expectedCluster, actual)
		})
	}
}

func TestFormatAltStatNameForPrometheus(t *testing.T) {
	testCases := []struct {
		name                string
		clusterName         string
		expectedAltStatName string
	}{
		{
			name:                "configure cluster name containing '.' and ':'",
			clusterName:         "foo.com:80",
			expectedAltStatName: "foo_com_80",
		},
		{
			name:                "configure cluster name containing multiple '.' and :''",
			clusterName:         "foo.bar.com:80",
			expectedAltStatName: "foo_bar_com_80",
		},
		{
			name:                "configure cluster name not containing '.' and ':'",
			clusterName:         "foo",
			expectedAltStatName: "foo",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := tassert.New(t)

			actual := formatAltStatNameForPrometheus(tc.clusterName)
			assert.Equal(tc.expectedAltStatName, actual)
		})
	}
}
