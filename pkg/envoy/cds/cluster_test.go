package cds

import (
	"net"
	"testing"

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
	"github.com/openservicemesh/osm/pkg/endpoint"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/tests"
)

func TestGetUpstreamServiceCluster(t *testing.T) {
	assert := tassert.New(t)

	mockCtrl := gomock.NewController(t)
	mockConfigurator := configurator.NewMockConfigurator(mockCtrl)
	downstreamSvc := tests.BookbuyerService
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
			remoteCluster, err := getUpstreamServiceCluster(upstreamSvc, downstreamSvc, mockConfigurator)
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
		name                        string
		endpoints                   []endpoint.Endpoint
		expectedLocalityLbEndpoints []*xds_endpoint.LocalityLbEndpoints
		expectedLbPolicy            xds_cluster.Cluster_LbPolicy
		expectedProtocolSelection   xds_cluster.Cluster_ClusterProtocolSelection
	}{
		{
			name:      "when service returns one endpoint",
			endpoints: []endpoint.Endpoint{tests.Endpoint},
			expectedLocalityLbEndpoints: []*xds_endpoint.LocalityLbEndpoints{
				{
					Locality: &xds_core.Locality{
						Zone: "zone",
					},
					LbEndpoints: []*xds_endpoint.LbEndpoint{{
						HostIdentifier: &xds_endpoint.LbEndpoint_Endpoint{
							Endpoint: &xds_endpoint.Endpoint{
								Address: envoy.GetAddress(constants.WildcardIPAddr, uint32(tests.ServicePort)),
							},
						},
						LoadBalancingWeight: &wrappers.UInt32Value{
							Value: constants.ClusterWeightAcceptAll, // Local cluster accepts all traffic
						},
					}},
				},
			},
		},
		{
			name: "when service returns two endpoints with same port",
			endpoints: []endpoint.Endpoint{tests.Endpoint, {
				IP:   net.ParseIP("1.2.3.4"),
				Port: endpoint.Port(tests.ServicePort),
			}},
			expectedLocalityLbEndpoints: []*xds_endpoint.LocalityLbEndpoints{
				{
					Locality: &xds_core.Locality{
						Zone: "zone",
					},
					LbEndpoints: []*xds_endpoint.LbEndpoint{{
						HostIdentifier: &xds_endpoint.LbEndpoint_Endpoint{
							Endpoint: &xds_endpoint.Endpoint{
								Address: envoy.GetAddress(constants.WildcardIPAddr, uint32(tests.ServicePort)),
							},
						},
						LoadBalancingWeight: &wrappers.UInt32Value{
							Value: constants.ClusterWeightAcceptAll, // Local cluster accepts all traffic
						},
					}},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCatalog.EXPECT().ListEndpointsForService(proxyService).Return(tc.endpoints, nil).Times(1)
			cluster, err := getLocalServiceCluster(mockCatalog, proxyService, clusterName)
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
			//assert.Equal(tc.expectedLocalityLbEndpoints, cluster.LoadAssignment.Endpoints)
		})
	}
}
