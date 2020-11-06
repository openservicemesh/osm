package cds

import (
	xds_cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/tests"
)

var _ = Describe("Cluster configurations", func() {
	var (
		mockCtrl         *gomock.Controller
		mockConfigurator *configurator.MockConfigurator
	)

	mockCtrl = gomock.NewController(GinkgoT())
	mockConfigurator = configurator.NewMockConfigurator(mockCtrl)

	downstreamSvc := tests.BookbuyerService
	upstreamSvc := tests.BookstoreV1Service
	Context("Test getUpstreamServiceCluster", func() {
		It("Returns an EDS based cluster when permissive mode is disabled", func() {
			mockConfigurator.EXPECT().IsPermissiveTrafficPolicyMode().Return(false).Times(1)

			remoteCluster, err := getUpstreamServiceCluster(upstreamSvc, downstreamSvc, mockConfigurator)
			Expect(err).ToNot(HaveOccurred())
			Expect(remoteCluster.GetType()).To(Equal(xds_cluster.Cluster_EDS))
			Expect(remoteCluster.LbPolicy).To(Equal(xds_cluster.Cluster_ROUND_ROBIN))
			Expect(remoteCluster.ProtocolSelection).To(Equal(xds_cluster.Cluster_USE_DOWNSTREAM_PROTOCOL))
		})

		It("Returns an Original Destination based cluster when permissive mode is enabled", func() {
			mockConfigurator.EXPECT().IsPermissiveTrafficPolicyMode().Return(true).Times(1)

			remoteCluster, err := getUpstreamServiceCluster(upstreamSvc, downstreamSvc, mockConfigurator)
			Expect(err).ToNot(HaveOccurred())
			Expect(remoteCluster.GetType()).To(Equal(xds_cluster.Cluster_ORIGINAL_DST))
			Expect(remoteCluster.LbPolicy).To(Equal(xds_cluster.Cluster_CLUSTER_PROVIDED))
			Expect(remoteCluster.ProtocolSelection).To(Equal(xds_cluster.Cluster_USE_DOWNSTREAM_PROTOCOL))
		})
	})
})
