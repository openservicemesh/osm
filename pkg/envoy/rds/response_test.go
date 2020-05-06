package rds

import (
	set "github.com/deckarep/golang-set"
	"github.com/open-service-mesh/osm/pkg/endpoint"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Route exists in routePolicyWeightedClustersList", func() {
	Context("Testing a route is already in a given list of routes", func() {
		It("Returns true and the index of route in the list", func() {

			weightedClusters := set.NewSetFromSlice([]interface{}{
				endpoint.WeightedCluster{ClusterName: endpoint.ClusterName("osm/bookstore-1"), Weight: 100},
				endpoint.WeightedCluster{ClusterName: endpoint.ClusterName("osm/bookstore-2"), Weight: 100},
			})
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

			weightedClusters := set.NewSetFromSlice([]interface{}{
				endpoint.WeightedCluster{ClusterName: endpoint.ClusterName("osm/bookstore-1"), Weight: 100},
				endpoint.WeightedCluster{ClusterName: endpoint.ClusterName("osm/bookstore-2"), Weight: 100},
			})
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
			Expect(routePolicyWeightedClusters.WeightedClusters.Cardinality()).To(Equal(1))
			routePolicyWeightedClustersSlice := routePolicyWeightedClusters.WeightedClusters.ToSlice()
			Expect(string(routePolicyWeightedClustersSlice[0].(endpoint.WeightedCluster).ClusterName)).To(Equal("osm/bookstore-1"))
			Expect(routePolicyWeightedClustersSlice[0].(endpoint.WeightedCluster).Weight).To(Equal(100))
		})
	})
})

var _ = Describe("AggregateRoutesByDomain", func() {
	domainRoutesMap := make(map[string][]endpoint.RoutePolicyWeightedClusters)
	weightedClustersMap := set.NewSet()
	Context("Building a map of routes by domain", func() {
		It("Returns a new aggregated map of domain and routes", func() {

			weightedCluster := endpoint.WeightedCluster{ClusterName: endpoint.ClusterName("osm/bookstore-1"), Weight: 100}
			routePolicies := []endpoint.RoutePolicy{
				{PathRegex: "/books-bought", Methods: []string{"GET"}},
				{PathRegex: "/buy-a-book", Methods: []string{"GET"}},
			}
			weightedClustersMap.Add(weightedCluster)

			aggregateRoutesByDomain(domainRoutesMap, routePolicies, weightedCluster, "bookstore.mesh")
			Expect(domainRoutesMap).NotTo(Equal(nil))
			Expect(len(domainRoutesMap)).To(Equal(1))
			Expect(len(domainRoutesMap["bookstore.mesh"])).To(Equal(2))

			for j := range domainRoutesMap["bookstore.mesh"] {
				Expect(domainRoutesMap["bookstore.mesh"][j].RoutePolicy).To(Equal(routePolicies[j]))
				Expect(domainRoutesMap["bookstore.mesh"][j].WeightedClusters.Cardinality()).To(Equal(1))
				Expect(domainRoutesMap["bookstore.mesh"][j].WeightedClusters.Equal(weightedClustersMap)).To(Equal(true))
			}
		})
	})

	Context("Adding a route to existing domain in the map", func() {
		It("Returns the map of with a new route on the domain", func() {

			weightedCluster := endpoint.WeightedCluster{ClusterName: endpoint.ClusterName("osm/bookstore-1"), Weight: 100}
			routePolicies := []endpoint.RoutePolicy{
				{PathRegex: "/update-books-bought", Methods: []string{"GET"}},
			}
			weightedClustersMap.Add(weightedCluster)

			aggregateRoutesByDomain(domainRoutesMap, routePolicies, weightedCluster, "bookstore.mesh")
			Expect(domainRoutesMap).NotTo(Equal(nil))
			Expect(len(domainRoutesMap)).To(Equal(1))
			Expect(len(domainRoutesMap["bookstore.mesh"])).To(Equal(3))
			Expect(domainRoutesMap["bookstore.mesh"][2].RoutePolicy).To(Equal(routePolicies[0]))
			Expect(domainRoutesMap["bookstore.mesh"][2].WeightedClusters.Cardinality()).To(Equal(1))
			Expect(domainRoutesMap["bookstore.mesh"][2].WeightedClusters.Equal(weightedClustersMap)).To(Equal(true))
		})
	})

	Context("Adding a cluster to an existing route to existing domain in the map", func() {
		It("Returns the map of with a new weighted cluster on a route in the domain", func() {

			weightedCluster := endpoint.WeightedCluster{ClusterName: endpoint.ClusterName("osm/bookstore-2"), Weight: 100}
			routePolicies := []endpoint.RoutePolicy{
				{PathRegex: "/update-books-bought", Methods: []string{"GET"}},
			}
			weightedClustersMap.Add(weightedCluster)

			aggregateRoutesByDomain(domainRoutesMap, routePolicies, weightedCluster, "bookstore.mesh")
			Expect(domainRoutesMap).NotTo(Equal(nil))
			Expect(len(domainRoutesMap)).To(Equal(1))
			Expect(len(domainRoutesMap["bookstore.mesh"])).To(Equal(3))
			Expect(domainRoutesMap["bookstore.mesh"][2].RoutePolicy).To(Equal(endpoint.RoutePolicy{PathRegex: "/update-books-bought", Methods: []string{"GET", "GET"}}))
			Expect(domainRoutesMap["bookstore.mesh"][2].WeightedClusters.Cardinality()).To(Equal(2))
			Expect(domainRoutesMap["bookstore.mesh"][2].WeightedClusters.Equal(weightedClustersMap)).To(Equal(true))
		})
	})
})
