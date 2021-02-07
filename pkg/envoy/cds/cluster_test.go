package cds

import (
	"testing"

	xds_cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	tassert "github.com/stretchr/testify/assert"

	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/tests"
)

func TestGetUpstreamServiceCluster(t *testing.T) {
	assert := tassert.New(t)

	mockCtrl := gomock.NewController(GinkgoT())
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
