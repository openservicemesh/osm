package rds

import (
	"github.com/open-service-mesh/osm/pkg/endpoint"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Route exists in routePolicyWeightedClustersList", func() {
	Context("Testing a route is already in a given list of routes", func() {
		It("Returns true and the index of route in the list", func() {

			weightedClusters := []endpoint.WeightedCluster{
				{ClusterName: endpoint.ClusterName("osm/bookstore-1"), Weight: 100},
				{ClusterName: endpoint.ClusterName("osm/bookstore-2"), Weight: 100},
			}
			routePolicy := endpoint.RoutePolicy{
				PathRegex: "/books-bought",
				Methods:   []string{"GET"},
			}
			routePolicyWeightedClustersList := []endpoint.RoutePolicyWeightedClusters{
				{RoutePolicy: routePolicy, WeightedClusters: weightedClusters},
			}
			newRoutePolicy := endpoint.RoutePolicy{
				PathRegex: "/books-bought",
				Methods:   []string{"GET"},
			}

			index, routeExists := routeExits(routePolicyWeightedClustersList, newRoutePolicy)
			Expect(index).To(Equal(0))
			Expect(routeExists).To(Equal(true))
		})
	})

	Context("Testing a route doesn't exist a given list of routes", func() {
		It("Returns false and the index of -1", func() {

			weightedClusters := []endpoint.WeightedCluster{
				{ClusterName: endpoint.ClusterName("osm/bookstore-1"), Weight: 100},
				{ClusterName: endpoint.ClusterName("osm/bookstore-2"), Weight: 100},
			}
			routePolicy := endpoint.RoutePolicy{
				PathRegex: "/books-bought",
				Methods:   []string{"GET"},
			}
			routeWeightedClustersList := []endpoint.RoutePolicyWeightedClusters{
				{RoutePolicy: routePolicy, WeightedClusters: weightedClusters},
			}
			newRoutePolicy := endpoint.RoutePolicy{
				PathRegex: "/buy-a-book",
				Methods:   []string{"GET"},
			}

			index, routeExists := routeExits(routeWeightedClustersList, newRoutePolicy)
			Expect(index).To(Equal(-1))
			Expect(routeExists).To(Equal(false))
		})
	})
})

var _ = Describe("Construct RoutePolicyWeightedClusters object", func() {
	Context("Testing the creating of a RoutePolicyWeightedClusters object", func() {
		It("Returns RoutePolicyWeightedClusters", func() {

			weightedCluster := endpoint.WeightedCluster{ClusterName: endpoint.ClusterName("osm/bookstore-1"), Weight: 100}
			routePolicy := endpoint.RoutePolicy{
				PathRegex: "/books-bought",
				Methods:   []string{"GET"},
			}

			routePolicyWeightedClusters := createRoutePolicyWeightedClusters(routePolicy, weightedCluster)
			Expect(routePolicyWeightedClusters).NotTo(Equal(nil))
			Expect(routePolicyWeightedClusters.RoutePolicy.PathRegex).To(Equal("/books-bought"))
			Expect(routePolicyWeightedClusters.RoutePolicy.Methods).To(Equal([]string{"GET"}))
			Expect(len(routePolicyWeightedClusters.WeightedClusters)).To(Equal(1))
			Expect(string(routePolicyWeightedClusters.WeightedClusters[0].ClusterName)).To(Equal("osm/bookstore-1"))
			Expect(routePolicyWeightedClusters.WeightedClusters[0].Weight).To(Equal(100))
		})
	})
})

var _ = Describe("AggregateRoutesByDomain", func() {
	domainRoutesMap := make(map[string][]endpoint.RoutePolicyWeightedClusters)
	Context("Building a map of routes by domain", func() {
		It("Returns a new aggregated map of domain and routes", func() {

			weightedCluster := endpoint.WeightedCluster{ClusterName: endpoint.ClusterName("osm/bookstore-1"), Weight: 100}
			routePolicies := []endpoint.RoutePolicy{
				{PathRegex: "/books-bought", Methods: []string{"GET"}},
				{PathRegex: "/buy-a-book", Methods: []string{"GET"}},
			}

			aggregateRoutesByDomain(domainRoutesMap, routePolicies, weightedCluster, "bookstore.mesh")
			Expect(domainRoutesMap).NotTo(Equal(nil))
			Expect(len(domainRoutesMap)).To(Equal(1))
			Expect(len(domainRoutesMap["bookstore.mesh"])).To(Equal(2))
			Expect(domainRoutesMap["bookstore.mesh"][0].RoutePolicy).To(Equal(endpoint.RoutePolicy{PathRegex: "/books-bought", Methods: []string{"GET"}}))
			Expect(len(domainRoutesMap["bookstore.mesh"][0].WeightedClusters)).To(Equal(1))
			Expect(domainRoutesMap["bookstore.mesh"][0].WeightedClusters[0]).To(Equal(endpoint.WeightedCluster{ClusterName: endpoint.ClusterName("osm/bookstore-1"), Weight: 100}))
			Expect(domainRoutesMap["bookstore.mesh"][1].RoutePolicy).To(Equal(endpoint.RoutePolicy{PathRegex: "/buy-a-book", Methods: []string{"GET"}}))
			Expect(len(domainRoutesMap["bookstore.mesh"][1].WeightedClusters)).To(Equal(1))
			Expect(domainRoutesMap["bookstore.mesh"][1].WeightedClusters[0]).To(Equal(endpoint.WeightedCluster{ClusterName: endpoint.ClusterName("osm/bookstore-1"), Weight: 100}))
		})
	})

	Context("Adding a route to existing domain in the map", func() {
		It("Returns the map of with a new route on the domain", func() {

			weightedCluster := endpoint.WeightedCluster{ClusterName: endpoint.ClusterName("osm/bookstore-1"), Weight: 100}
			routePolicies := []endpoint.RoutePolicy{
				{PathRegex: "/update-books-bought", Methods: []string{"GET"}},
			}

			aggregateRoutesByDomain(domainRoutesMap, routePolicies, weightedCluster, "bookstore.mesh")
			Expect(domainRoutesMap).NotTo(Equal(nil))
			Expect(len(domainRoutesMap)).To(Equal(1))
			Expect(len(domainRoutesMap["bookstore.mesh"])).To(Equal(3))
			Expect(domainRoutesMap["bookstore.mesh"][2].RoutePolicy).To(Equal(endpoint.RoutePolicy{PathRegex: "/update-books-bought", Methods: []string{"GET"}}))
			Expect(len(domainRoutesMap["bookstore.mesh"][2].WeightedClusters)).To(Equal(1))
			Expect(domainRoutesMap["bookstore.mesh"][0].WeightedClusters[0]).To(Equal(endpoint.WeightedCluster{ClusterName: endpoint.ClusterName("osm/bookstore-1"), Weight: 100}))
		})
	})

	Context("Adding a cluster to an existing route to existing domain in the map", func() {
		It("Returns the map of with a new weighted cluster on a route in the domain", func() {

			weightedCluster := endpoint.WeightedCluster{ClusterName: endpoint.ClusterName("osm/bookstore-2"), Weight: 100}
			routePolicies := []endpoint.RoutePolicy{
				{PathRegex: "/update-books-bought", Methods: []string{"GET"}},
			}

			aggregateRoutesByDomain(domainRoutesMap, routePolicies, weightedCluster, "bookstore.mesh")
			Expect(domainRoutesMap).NotTo(Equal(nil))
			Expect(len(domainRoutesMap)).To(Equal(1))
			Expect(len(domainRoutesMap["bookstore.mesh"])).To(Equal(3))
			Expect(domainRoutesMap["bookstore.mesh"][2].RoutePolicy).To(Equal(endpoint.RoutePolicy{PathRegex: "/update-books-bought", Methods: []string{"GET", "GET"}}))
			Expect(len(domainRoutesMap["bookstore.mesh"][2].WeightedClusters)).To(Equal(2))
			Expect(domainRoutesMap["bookstore.mesh"][2].WeightedClusters[0]).To(Equal(endpoint.WeightedCluster{ClusterName: endpoint.ClusterName("osm/bookstore-1"), Weight: 100}))
			Expect(domainRoutesMap["bookstore.mesh"][2].WeightedClusters[1]).To(Equal(endpoint.WeightedCluster{ClusterName: endpoint.ClusterName("osm/bookstore-2"), Weight: 100}))
		})
	})
})
