package cds

import (
	xds "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/open-service-mesh/osm/pkg/endpoint"
	"github.com/open-service-mesh/osm/pkg/envoy"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var uniqueness = Describe("UniqueLists", func() {
	Context("Testing uniqueness of clusters", func() {
		It("Returns unique list of clusterd for CDS", func() {

			clusters := []xds.Cluster{
				envoy.GetServiceCluster("osm/bookstore-1", endpoint.NamespacedService{Namespace: "osm", Service: "bookstore-1"}),
				envoy.GetServiceCluster("osm/bookstore-2", endpoint.NamespacedService{Namespace: "osm", Service: "bookstore-2"}),
				envoy.GetServiceCluster("osm/bookstore-1", endpoint.NamespacedService{Namespace: "osm", Service: "bookstore-1"}),
				getServiceClusterLocal(string("osm/bookstore-1" + envoy.LocalClusterSuffix)),
				getServiceClusterLocal(string("osm/bookstore-1" + envoy.LocalClusterSuffix)),
				getServiceClusterLocal(string("osm/bookstore-2" + envoy.LocalClusterSuffix)),
			}

			actualClusters := uniques(clusters)
			expectedClusters := []xds.Cluster{
				envoy.GetServiceCluster("osm/bookstore-1", endpoint.NamespacedService{Namespace: "osm", Service: "bookstore-1"}),
				envoy.GetServiceCluster("osm/bookstore-2", endpoint.NamespacedService{Namespace: "osm", Service: "bookstore-2"}),
				getServiceClusterLocal(string("osm/bookstore-1" + envoy.LocalClusterSuffix)),
				getServiceClusterLocal(string("osm/bookstore-2" + envoy.LocalClusterSuffix)),
			}

			Expect(actualClusters).To(Equal(expectedClusters))
		})
	})
})
