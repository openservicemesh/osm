package cds

import (
	xds_cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/tests"
)

var _ = Describe("Cluster configurations", func() {

	localService := tests.BookbuyerService
	remoteService := tests.BookstoreService
	Context("Test getRemoteServiceCluster", func() {
		It("Returns an EDS based cluster when permissive mode is disabled", func() {
			cfg := configurator.NewFakeConfiguratorWithOptions(
				configurator.FakeConfigurator{
					PermissiveTrafficPolicyMode: false,
				},
			)

			remoteCluster, err := getRemoteServiceCluster(remoteService, localService, cfg)
			Expect(err).ToNot(HaveOccurred())
			Expect(remoteCluster.GetType()).To(Equal(xds_cluster.Cluster_EDS))
			Expect(remoteCluster.LbPolicy).To(Equal(xds_cluster.Cluster_ROUND_ROBIN))
			Expect(remoteCluster.ProtocolSelection).To(Equal(xds_cluster.Cluster_USE_DOWNSTREAM_PROTOCOL))
		})

		It("Returns an Original Destination based cluster when permissive mode is enabled", func() {
			cfg := configurator.NewFakeConfiguratorWithOptions(
				configurator.FakeConfigurator{
					PermissiveTrafficPolicyMode: true,
				},
			)

			remoteCluster, err := getRemoteServiceCluster(remoteService, localService, cfg)
			Expect(err).ToNot(HaveOccurred())
			Expect(remoteCluster.GetType()).To(Equal(xds_cluster.Cluster_ORIGINAL_DST))
			Expect(remoteCluster.LbPolicy).To(Equal(xds_cluster.Cluster_CLUSTER_PROVIDED))
			Expect(remoteCluster.ProtocolSelection).To(Equal(xds_cluster.Cluster_USE_DOWNSTREAM_PROTOCOL))
		})
	})
})
