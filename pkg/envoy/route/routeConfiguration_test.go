package route

import (
	set "github.com/deckarep/golang-set"
	v2route "github.com/envoyproxy/go-control-plane/envoy/api/v2/route"
	"github.com/golang/protobuf/ptypes/wrappers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/open-service-mesh/osm/pkg/constants"
	"github.com/open-service-mesh/osm/pkg/envoy"
	"github.com/open-service-mesh/osm/pkg/service"
	"github.com/open-service-mesh/osm/pkg/trafficpolicy"
)

var _ = Describe("VirtualHost with domains", func() {
	Context("Testing the creation of virtual host object in the route configuration", func() {
		It("Returns the virtual host with domains", func() {
			virtualHost := createVirtualHostStub("inboud|bookstore.mesh", "bookstore.mesh")
			Expect(virtualHost.Name).To(Equal("inboud|bookstore.mesh"))
			Expect(virtualHost.Domains).To(Equal([]string{"bookstore.mesh"}))
			Expect(len(virtualHost.Domains)).To(Equal(1))
			Expect(len(virtualHost.Routes)).To(Equal(0))
		})
	})
})

var _ = Describe("Allowed methods on a route", func() {
	Context("Testing sanitizeHTTPMethods", func() {
		It("Returns a unique list of allowed methods", func() {

			allowedMethods := []string{"GET", "POST", "PUT", "POST", "GET", "GET"}
			allowedMethods = sanitizeHTTPMethods(allowedMethods)

			expectedAllowedMethods := []string{"GET", "POST", "PUT"}
			Expect(allowedMethods).To(Equal(expectedAllowedMethods))

		})

		It("Returns a wildcard allowed method (*)", func() {
			allowedMethods := []string{"GET", "POST", "PUT", "POST", "GET", "GET", "*"}
			allowedMethods = sanitizeHTTPMethods(allowedMethods)

			expectedAllowedMethods := []string{"*"}
			Expect(allowedMethods).To(Equal(expectedAllowedMethods))
		})
	})
})

var _ = Describe("Weighted clusters", func() {
	Context("Testing getWeightedClusters", func() {
		It("validated the creation of weighted clusters", func() {

			weightedClusters := set.NewSetFromSlice([]interface{}{
				service.WeightedCluster{ClusterName: service.ClusterName("osm/bookstore-1"), Weight: 30},
				service.WeightedCluster{ClusterName: service.ClusterName("osm/bookstore-2"), Weight: 70},
			})

			clustersExpected := set.NewSet()
			weightsExpected := set.NewSet()
			for weightedClusterInterface := range weightedClusters.Iter() {
				cluster := weightedClusterInterface.(service.WeightedCluster)
				clustersExpected.Add(string(cluster.ClusterName + envoy.LocalClusterSuffix))
				weightsExpected.Add(uint32(cluster.Weight))
			}

			totalClusterWeight := 0
			for clusterInterface := range weightedClusters.Iter() {
				cluster := clusterInterface.(service.WeightedCluster)
				totalClusterWeight += cluster.Weight
			}

			routeWeightedClusters := getWeightedCluster(weightedClusters, totalClusterWeight, true)
			Expect(routeWeightedClusters.TotalWeight).To(Equal(&wrappers.UInt32Value{Value: uint32(totalClusterWeight)}))
			Expect(len(routeWeightedClusters.GetClusters())).To(Equal(weightedClusters.Cardinality()))

			clustersActual := set.NewSet()
			weightsActual := set.NewSet()
			for _, cluster := range routeWeightedClusters.GetClusters() {
				clustersActual.Add(cluster.Name)
				weightsActual.Add(cluster.Weight.Value)
			}
			Expect(clustersActual.Equal(clustersExpected)).To(Equal(true))
			Expect(weightsActual.Equal(weightsExpected)).To(Equal(true))
		})
	})
})

var _ = Describe("Routes with weighted clusters", func() {
	Context("Testing creation of routes object in route configuration", func() {
		routeWeightedClustersMap := make(map[string]trafficpolicy.RouteWeightedClusters)
		weightedClusters := set.NewSetFromSlice([]interface{}{
			service.WeightedCluster{ClusterName: service.ClusterName("osm/bookstore-1"), Weight: 100},
			service.WeightedCluster{ClusterName: service.ClusterName("osm/bookstore-2"), Weight: 100},
		})

		clustersExpected := set.NewSet()
		weightsExpected := set.NewSet()
		for weightedClusterInterface := range weightedClusters.Iter() {
			cluster := weightedClusterInterface.(service.WeightedCluster)
			clustersExpected.Add(string(cluster.ClusterName + envoy.LocalClusterSuffix))
			weightsExpected.Add(uint32(cluster.Weight))
		}

		totalClusterWeight := 0
		for clusterInterface := range weightedClusters.Iter() {
			cluster := clusterInterface.(service.WeightedCluster)
			totalClusterWeight += cluster.Weight
		}

		clustersActual := set.NewSet()
		weightsActual := set.NewSet()

		It("Adds a new route", func() {

			routePolicy := trafficpolicy.Route{
				PathRegex: "/books-bought",
				Methods:   []string{"GET", "POST"},
			}

			routeWeightedClustersMap[routePolicy.PathRegex] = trafficpolicy.RouteWeightedClusters{Route: routePolicy, WeightedClusters: weightedClusters}
			rt := createRoutes(routeWeightedClustersMap, true)
			Expect(len(rt)).To(Equal(len(routePolicy.Methods)))

			for i, route := range rt {
				Expect(route.Match.GetSafeRegex().Regex).To(Equal(routePolicy.PathRegex))
				Expect(route.Match.GetHeaders()[0].GetSafeRegexMatch().Regex).To(Equal(routePolicy.Methods[i]))
				Expect(route.GetRoute().GetWeightedClusters().TotalWeight).To(Equal(&wrappers.UInt32Value{Value: uint32(totalClusterWeight)}))
				Expect(len(route.GetRoute().GetWeightedClusters().GetClusters())).To(Equal(weightedClusters.Cardinality()))
				for _, cluster := range route.GetRoute().GetWeightedClusters().GetClusters() {
					clustersActual.Add(cluster.Name)
					weightsActual.Add(cluster.Weight.Value)
				}
			}
			Expect(clustersActual.Equal(clustersExpected)).To(Equal(true))
			Expect(weightsActual.Equal(weightsExpected)).To(Equal(true))
		})

		It("Appends another route", func() {

			routePolicy2 := trafficpolicy.Route{
				PathRegex: "/buy-a-book",
				Methods:   []string{"GET"},
			}
			routeWeightedClustersMap[routePolicy2.PathRegex] = trafficpolicy.RouteWeightedClusters{Route: routePolicy2, WeightedClusters: weightedClusters}

			httpMethodCount := 3 // 2 from previously added routes + 1 append

			rt := createRoutes(routeWeightedClustersMap, true)

			Expect(len(rt)).To(Equal(httpMethodCount))
			var newRoute *v2route.Route
			for _, route := range rt {
				if route.Match.GetSafeRegex().Regex == routePolicy2.PathRegex {
					newRoute = route
				}
			}
			Expect(newRoute.Match.GetHeaders()[0].GetSafeRegexMatch().Regex).To(Equal(routePolicy2.Methods[0]))
			Expect(newRoute.GetRoute().GetWeightedClusters().TotalWeight).To(Equal(&wrappers.UInt32Value{Value: uint32(totalClusterWeight)}))
			Expect(len(newRoute.GetRoute().GetWeightedClusters().GetClusters())).To(Equal(weightedClusters.Cardinality()))
			for _, cluster := range newRoute.GetRoute().GetWeightedClusters().GetClusters() {
				clustersActual.Add(cluster.Name)
				weightsActual.Add(cluster.Weight.Value)
			}
			Expect(clustersActual.Equal(clustersExpected)).To(Equal(true))
			Expect(weightsActual.Equal(weightsExpected)).To(Equal(true))
		})
	})
})

var _ = Describe("Route Configuration", func() {
	Context("Testing creation of RouteConfiguration object", func() {
		It("Returns outbound route configuration", func() {

			weightedClusters := set.NewSet()
			weightedClusters.Add(service.WeightedCluster{ClusterName: service.ClusterName("osm/bookstore-1"), Weight: 100})
			weightedClusters.Add(service.WeightedCluster{ClusterName: service.ClusterName("osm/bookstore-2"), Weight: 100})

			totalClusterWeight := 0
			for clusterInterface := range weightedClusters.Iter() {
				cluster := clusterInterface.(service.WeightedCluster)
				totalClusterWeight += cluster.Weight
			}
			routePolicy := trafficpolicy.Route{
				PathRegex: "/books-bought",
				Methods:   []string{"GET"},
			}

			sourceDomainRouteData := map[string]trafficpolicy.RouteWeightedClusters{
				routePolicy.PathRegex: trafficpolicy.RouteWeightedClusters{Route: routePolicy, WeightedClusters: weightedClusters},
			}

			sourceDomainAggregatedData := map[string]map[string]trafficpolicy.RouteWeightedClusters{
				"bookstore.mesh": sourceDomainRouteData,
			}

			//Validating the outbound clusters and routes
			sourceRouteConfig := NewRouteConfigurationStub(OutboundRouteConfig)
			sourceRouteConfig = UpdateRouteConfiguration(sourceDomainAggregatedData, sourceRouteConfig, true, false)
			Expect(sourceRouteConfig).NotTo(Equal(nil))
			Expect(sourceRouteConfig.Name).To(Equal(OutboundRouteConfig))
			Expect(len(sourceRouteConfig.VirtualHosts)).To(Equal(len(sourceDomainAggregatedData)))
			Expect(len(sourceRouteConfig.VirtualHosts[0].Routes)).To(Equal(1))
			Expect(sourceRouteConfig.VirtualHosts[0].Routes[0].Match.GetSafeRegex().Regex).To(Equal(constants.RegexMatchAll))
			Expect(sourceRouteConfig.VirtualHosts[0].Routes[0].Match.GetHeaders()[0].GetSafeRegexMatch().Regex).To(Equal(constants.RegexMatchAll))
			Expect(len(sourceRouteConfig.VirtualHosts[0].Routes[0].GetRoute().GetWeightedClusters().GetClusters())).To(Equal(weightedClusters.Cardinality()))
			Expect(sourceRouteConfig.VirtualHosts[0].Routes[0].GetRoute().GetWeightedClusters().TotalWeight).To(Equal(&wrappers.UInt32Value{Value: uint32(totalClusterWeight)}))
		})
	})
})

var _ = Describe("Route Configuration", func() {
	Context("Testing regex matches for HTTP methods", func() {
		It("Tests that the wildcard HTTP method correctly translates to a match all regex", func() {
			regex := getRegexForMethod("*")
			Expect(regex).To(Equal(constants.RegexMatchAll))
		})
		It("Tests that a non wildcard HTTP method correctly translates to its corresponding regex", func() {
			regex := getRegexForMethod("GET")
			Expect(regex).To(Equal("GET"))
		})
	})
})
