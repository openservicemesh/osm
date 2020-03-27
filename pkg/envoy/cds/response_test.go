package cds

import (
	xds "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/open-service-mesh/osm/pkg/endpoint"
	"github.com/open-service-mesh/osm/pkg/envoy"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("UniqueLists", func() {
	Context("Testing uniqueness of clusters", func() {
		It("Returns unique list of clusters for CDS", func() {

			// Create xds.cluster objects, some having the same cluster name
			clusters := []xds.Cluster{
				envoy.GetServiceCluster("osm/bookstore-1", endpoint.NamespacedService{Namespace: "osm", Service: "bookstore-1"}),
				envoy.GetServiceCluster("osm/bookstore-2", endpoint.NamespacedService{Namespace: "osm", Service: "bookstore-2"}),
				envoy.GetServiceCluster("osm/bookstore-1", endpoint.NamespacedService{Namespace: "osm", Service: "bookstore-1"}),
			}

			// Filter out xds.Cluster objects having the same name
			actualClusters := uniques(clusters)
			expectedClusters := []xds.Cluster{
				envoy.GetServiceCluster("osm/bookstore-1", endpoint.NamespacedService{Namespace: "osm", Service: "bookstore-1"}),
				envoy.GetServiceCluster("osm/bookstore-2", endpoint.NamespacedService{Namespace: "osm", Service: "bookstore-2"}),
			}

			Expect(actualClusters).To(Equal(expectedClusters))
		})
	})
})
