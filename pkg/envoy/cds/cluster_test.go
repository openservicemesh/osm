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
)

func TestGetUpstreamServiceCluster(t *testing.T) {
	assert := tassert.New(t)

	mockCtrl := gomock.NewController(t)
	mockConfigurator := configurator.NewMockConfigurator(mockCtrl)

	downstreamSvcAccount := tests.BookbuyerServiceAccount
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

func TestGetSyntheticCluster(t *testing.T) {
	assert := tassert.New(t)

	actual := getSyntheticCluster("foo")
	assert.NotNil(actual)
	assert.Equal("foo", actual.Name)
	assert.Equal(&xds_cluster.Cluster_Type{Type: xds_cluster.Cluster_STATIC}, actual.ClusterDiscoveryType)
	assert.Equal(xds_cluster.Cluster_ROUND_ROBIN, actual.LbPolicy)
	assert.Equal(ptypes.DurationProto(clusterConnectTimeout), actual.ConnectTimeout)
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

func TestGetOutboundPassthroughCluster(t *testing.T) {
	assert := tassert.New(t)

	expectedCluster := &xds_cluster.Cluster{
		Name:           envoy.OutboundPassthroughCluster,
		ConnectTimeout: ptypes.DurationProto(1 * time.Second),
		ClusterDiscoveryType: &xds_cluster.Cluster_Type{
			Type: xds_cluster.Cluster_ORIGINAL_DST,
		},
		LbPolicy:             xds_cluster.Cluster_CLUSTER_PROVIDED,
		ProtocolSelection:    xds_cluster.Cluster_USE_DOWNSTREAM_PROTOCOL,
		Http2ProtocolOptions: &xds_core.Http2ProtocolOptions{},
	}

	actual := getOutboundPassthroughCluster()

	assert.Equal(expectedCluster, actual)
}
