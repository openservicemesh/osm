package cds

import (
	"github.com/open-service-mesh/osm/pkg/endpoint"
	"github.com/open-service-mesh/osm/pkg/envoy"
	xds "github.com/envoyproxy/go-control-plane/envoy/api/v2"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var uniqueness = Describe("UniqueLists", func() {
	Context("Testing uniqueness of clusters", func() {
		It("Returns unique list of clusterd for CDS", func() {

			clusters := []xds.Cluster{
				envoy.GetServiceCluster("smc/bookstore-1", endpoint.NamespacedService{Namespace: "smc", Service: "bookstore-1"}),
				envoy.GetServiceCluster("smc/bookstore-2", endpoint.NamespacedService{Namespace: "smc", Service: "bookstore-2"}),
				envoy.GetServiceCluster("smc/bookstore-1", endpoint.NamespacedService{Namespace: "smc", Service: "bookstore-1"}),
				getServiceClusterLocal(string("smc/bookstore-1" + envoy.LocalClusterSuffix)),
				getServiceClusterLocal(string("smc/bookstore-1" + envoy.LocalClusterSuffix)),
				getServiceClusterLocal(string("smc/bookstore-2" + envoy.LocalClusterSuffix)),
			}

			actualClusters := uniques(clusters)
			expectedClusters := []xds.Cluster{
				envoy.GetServiceCluster("smc/bookstore-1", endpoint.NamespacedService{Namespace: "smc", Service: "bookstore-1"}),
				envoy.GetServiceCluster("smc/bookstore-2", endpoint.NamespacedService{Namespace: "smc", Service: "bookstore-2"}),
				getServiceClusterLocal(string("smc/bookstore-1" + envoy.LocalClusterSuffix)),
				getServiceClusterLocal(string("smc/bookstore-2" + envoy.LocalClusterSuffix)),
			}

			Expect(actualClusters).To(Equal(expectedClusters))
		})
	})
})
