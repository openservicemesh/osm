package catalog

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/open-service-mesh/osm/pkg/endpoint"
)

var uniqueness = Describe("UniqueLists", func() {
	Context("Testing uniqueness of services", func() {
		It("Returns unique list of services", func() {

			services := []endpoint.NamespacedService{
				endpoint.NamespacedService{Namespace: "osm", Service: "booktore-1"},
				endpoint.NamespacedService{Namespace: "osm", Service: "booktore-1"},
				endpoint.NamespacedService{Namespace: "osm", Service: "booktore-2"},
				endpoint.NamespacedService{Namespace: "osm", Service: "booktore-3"},
				endpoint.NamespacedService{Namespace: "osm", Service: "booktore-2"},
			}

			actual := uniqueServices(services)
			expected := []endpoint.NamespacedService{
				endpoint.NamespacedService{Namespace: "osm", Service: "booktore-1"},
				endpoint.NamespacedService{Namespace: "osm", Service: "booktore-2"},
				endpoint.NamespacedService{Namespace: "osm", Service: "booktore-3"},
			}
			Expect(actual).To(Equal(expected))
		})
	})

	Context("Testing uniqueness of weighted clusters", func() {
		It("Returns unique list of weighted clusters", func() {

			weightedClusters := []endpoint.WeightedCluster{
				{ClusterName: endpoint.ClusterName("osm/bookstore-1"), Weight: 100},
				{ClusterName: endpoint.ClusterName("osm/bookstore-1-local"), Weight: 100},
				{ClusterName: endpoint.ClusterName("osm/bookstore-1-local"), Weight: 100},
				{ClusterName: endpoint.ClusterName("osm/bookstore-2"), Weight: 100},
				{ClusterName: endpoint.ClusterName("osm/bookstore-2"), Weight: 50},
			}

			actual := uniqueClusters(weightedClusters)
			expected := []endpoint.WeightedCluster{
				{ClusterName: endpoint.ClusterName("osm/bookstore-1"), Weight: 100},
				{ClusterName: endpoint.ClusterName("osm/bookstore-1-local"), Weight: 100},
				{ClusterName: endpoint.ClusterName("osm/bookstore-2"), Weight: 100},
				{ClusterName: endpoint.ClusterName("osm/bookstore-2"), Weight: 50},
			}
			Expect(actual).To(Equal(expected))
		})
	})
})

var serviceToString = Describe("ServicesToString", func() {
	Context("Testing servicesToString", func() {
		It("Returns string list", func() {

			services := []endpoint.NamespacedService{
				endpoint.NamespacedService{Namespace: "osm", Service: "bookstore-1"},
				endpoint.NamespacedService{Namespace: "osm", Service: "bookstore-2"},
			}

			actual := servicesToString(services)
			expected := []string{
				"osm/bookstore-1",
				"osm/bookstore-2",
			}
			Expect(actual).To(Equal(expected))
		})
	})
})
