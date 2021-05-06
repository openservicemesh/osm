package cds

import (
	"errors"
	"testing"
	"time"

	xds_cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	xds_core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	xds_endpoint "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	"github.com/golang/mock/gomock"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/wrappers"
	tassert "github.com/stretchr/testify/assert"

	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/tests"
	"github.com/openservicemesh/osm/pkg/trafficpolicy"
)

func TestGetUpstreamServiceCluster(t *testing.T) {
	assert := tassert.New(t)

	mockCtrl := gomock.NewController(t)
	mockConfigurator := configurator.NewMockConfigurator(mockCtrl)

	downstreamSvcAccount := tests.BookbuyerServiceIdentity
	upstreamSvc := tests.BookstoreV1Service

	testCases := []struct {
		name                      string
		permissiveMode            bool
		expectedClusterType       xds_cluster.Cluster_DiscoveryType
		expectedLbPolicy          xds_cluster.Cluster_LbPolicy
		expectedProtocolSelection xds_cluster.Cluster_ClusterProtocolSelection
	}{
		{
			name:                      "Returns an EDS based cluster when permissive mode is disabled",
			permissiveMode:            false,
			expectedClusterType:       xds_cluster.Cluster_EDS,
			expectedLbPolicy:          xds_cluster.Cluster_ROUND_ROBIN,
			expectedProtocolSelection: xds_cluster.Cluster_USE_DOWNSTREAM_PROTOCOL,
		},
		{
			name:                      "Returns an Original Destination based cluster when permissive mode is enabled",
			permissiveMode:            true,
			expectedClusterType:       xds_cluster.Cluster_ORIGINAL_DST,
			expectedLbPolicy:          xds_cluster.Cluster_CLUSTER_PROVIDED,
			expectedProtocolSelection: xds_cluster.Cluster_USE_DOWNSTREAM_PROTOCOL,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockConfigurator.EXPECT().IsPermissiveTrafficPolicyMode().Return(tc.permissiveMode).Times(1)
			remoteCluster, err := getUpstreamServiceCluster(downstreamSvcAccount, upstreamSvc, mockConfigurator)
			assert.Nil(err)
			assert.Equal(tc.expectedClusterType, remoteCluster.GetType())
			assert.Equal(tc.expectedLbPolicy, remoteCluster.LbPolicy)
			assert.Equal(tc.expectedProtocolSelection, remoteCluster.ProtocolSelection)
		})
	}
}

func TestGetLocalServiceCluster(t *testing.T) {
	assert := tassert.New(t)

	clusterName := "bookbuyer-local"
	proxyService := service.MeshService{
		Name:      "bookbuyer",
		Namespace: "bookbuyer-ns",
	}

	mockCtrl := gomock.NewController(t)
	mockCatalog := catalog.NewMockMeshCataloger(mockCtrl)

	testCases := []struct {
		name                             string
		proxyService                     service.MeshService
		portToProtocolMapping            map[uint32]string
		expectedLocalityLbEndpoints      []*xds_endpoint.LocalityLbEndpoints
		expectedLbPolicy                 xds_cluster.Cluster_LbPolicy
		expectedProtocolSelection        xds_cluster.Cluster_ClusterProtocolSelection
		expectedPortToProtocolMappingErr bool
		expectedErr                      bool
	}{
		{
			name:                  "when service returns a single port",
			proxyService:          proxyService,
			portToProtocolMapping: map[uint32]string{uint32(8080): "something"},
			expectedLocalityLbEndpoints: []*xds_endpoint.LocalityLbEndpoints{
				{
					Locality: &xds_core.Locality{
						Zone: "zone",
					},
					LbEndpoints: []*xds_endpoint.LbEndpoint{{
						HostIdentifier: &xds_endpoint.LbEndpoint_Endpoint{
							Endpoint: &xds_endpoint.Endpoint{
								Address: envoy.GetAddress(constants.WildcardIPAddr, uint32(8080)),
							},
						},
						LoadBalancingWeight: &wrappers.UInt32Value{
							Value: constants.ClusterWeightAcceptAll, // Local cluster accepts all traffic
						},
					}},
				},
			},
			expectedPortToProtocolMappingErr: false,
			expectedErr:                      false,
		},
		{
			name:                             "when err fetching ports",
			proxyService:                     proxyService,
			portToProtocolMapping:            map[uint32]string{},
			expectedLocalityLbEndpoints:      []*xds_endpoint.LocalityLbEndpoints{},
			expectedPortToProtocolMappingErr: true,
			expectedErr:                      true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.expectedPortToProtocolMappingErr {
				mockCatalog.EXPECT().GetTargetPortToProtocolMappingForService(tc.proxyService).Return(tc.portToProtocolMapping, errors.New("error")).Times(1)
			} else {
				mockCatalog.EXPECT().GetTargetPortToProtocolMappingForService(tc.proxyService).Return(tc.portToProtocolMapping, nil).Times(1)
			}

			cluster, err := getLocalServiceCluster(mockCatalog, tc.proxyService, clusterName)

			if tc.expectedErr {
				assert.NotNil(err)
				assert.Nil(cluster)
			} else {
				assert.Nil(err)
				assert.Equal(clusterName, cluster.Name)
				assert.Equal(clusterName, cluster.AltStatName)
				assert.Equal(ptypes.DurationProto(clusterConnectTimeout), cluster.ConnectTimeout)
				assert.Equal(xds_cluster.Cluster_ROUND_ROBIN, cluster.LbPolicy)
				assert.Equal(&xds_cluster.Cluster_Type{Type: xds_cluster.Cluster_STRICT_DNS}, cluster.ClusterDiscoveryType)
				assert.Equal(true, cluster.RespectDnsTtl)
				assert.Equal(xds_cluster.Cluster_V4_ONLY, cluster.DnsLookupFamily)
				assert.Equal(xds_cluster.Cluster_USE_DOWNSTREAM_PROTOCOL, cluster.ProtocolSelection)
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
		ConnectTimeout:         ptypes.DurationProto(1 * time.Second),
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

	testCases := []struct {
		name        string
		clusterName string
		expected    *xds_cluster.Cluster
	}{
		{
			name:        "foo cluster",
			clusterName: "foo",
			expected: &xds_cluster.Cluster{
				Name:           "foo",
				ConnectTimeout: ptypes.DurationProto(1 * time.Second),
				ClusterDiscoveryType: &xds_cluster.Cluster_Type{
					Type: xds_cluster.Cluster_ORIGINAL_DST,
				},
				LbPolicy:             xds_cluster.Cluster_CLUSTER_PROVIDED,
				ProtocolSelection:    xds_cluster.Cluster_USE_DOWNSTREAM_PROTOCOL,
				Http2ProtocolOptions: &xds_core.Http2ProtocolOptions{},
			},
		},
		{
			name:        "bar cluster",
			clusterName: "bar",
			expected: &xds_cluster.Cluster{
				Name:           "bar",
				ConnectTimeout: ptypes.DurationProto(1 * time.Second),
				ClusterDiscoveryType: &xds_cluster.Cluster_Type{
					Type: xds_cluster.Cluster_ORIGINAL_DST,
				},
				LbPolicy:             xds_cluster.Cluster_CLUSTER_PROVIDED,
				ProtocolSelection:    xds_cluster.Cluster_USE_DOWNSTREAM_PROTOCOL,
				Http2ProtocolOptions: &xds_core.Http2ProtocolOptions{},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := getOriginalDestinationEgressCluster(tc.clusterName)
			assert.Equal(tc.expected, actual)
		})
	}
}

func TestGetEgressClusters(t *testing.T) {
	assert := tassert.New(t)

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
			actual := getEgressClusters(tc.clusterConfigs)
			assert.Len(actual, tc.expectedClusterCount)
		})
	}
}

func TestGetDNSResolvableEgressCluster(t *testing.T) {
	assert := tassert.New(t)

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
				Name:           "foo.com:80",
				AltStatName:    "foo.com:80",
				ConnectTimeout: ptypes.DurationProto(clusterConnectTimeout),
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
			actual, err := getDNSResolvableEgressCluster(tc.clusterConfig)
			assert.Equal(tc.expectError, err != nil)
			assert.Equal(tc.expectedCluster, actual)
		})
	}
}
