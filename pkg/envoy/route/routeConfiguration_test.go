package route

import (
	"fmt"

	"github.com/golang/protobuf/ptypes/wrappers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/open-service-mesh/osm/pkg/constants"
	"github.com/open-service-mesh/osm/pkg/endpoint"
	"github.com/open-service-mesh/osm/pkg/envoy"
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

			weightedClusters := []endpoint.WeightedCluster{
				{ClusterName: endpoint.ClusterName("osm/bookstore-1"), Weight: 30},
				{ClusterName: endpoint.ClusterName("osm/bookstore-2"), Weight: 70},
			}
			totalClusterWeight := 0
			for _, cluster := range weightedClusters {
				totalClusterWeight += cluster.Weight
			}

			routeWeightedClusters := getWeightedCluster(weightedClusters, totalClusterWeight, true)
			Expect(routeWeightedClusters.TotalWeight).To(Equal(&wrappers.UInt32Value{Value: uint32(totalClusterWeight)}))
			Expect(len(routeWeightedClusters.GetClusters())).To(Equal(len(weightedClusters)))

			for j, cluster := range routeWeightedClusters.GetClusters() {
				Expect(cluster.Name).To(Equal(fmt.Sprintf("%s-local", weightedClusters[j].ClusterName)))
				Expect(cluster.Weight).To(Equal(&wrappers.UInt32Value{Value: uint32(weightedClusters[j].Weight)}))
			}
		})
	})
})

var _ = Describe("Routes with weighted clusters", func() {
	Context("Testing creation of routes object in route configuration", func() {
		var routeWeightedClustersList []endpoint.RoutePolicyWeightedClusters
		weightedClusters := []endpoint.WeightedCluster{
			{ClusterName: endpoint.ClusterName("osm/bookstore-1"), Weight: 100},
			{ClusterName: endpoint.ClusterName("osm/bookstore-2"), Weight: 100},
		}
		totalClusterWeight := 0
		for _, cluster := range weightedClusters {
			totalClusterWeight += cluster.Weight
		}

		It("Adds a new route", func() {

			routePolicy := endpoint.RoutePolicy{
				PathRegex: "/books-bought",
				Methods:   []string{"GET", "POST"},
			}
			routeWeightedClustersList = append(routeWeightedClustersList, endpoint.RoutePolicyWeightedClusters{RoutePolicy: routePolicy, WeightedClusters: weightedClusters})

			rt := createRoutes(routeWeightedClustersList, true)
			Expect(len(rt)).To(Equal(len(routePolicy.Methods)))

			for i, route := range rt {
				Expect(route.Match.GetSafeRegex().Regex).To(Equal(routePolicy.PathRegex))
				Expect(route.Match.GetHeaders()[0].GetSafeRegexMatch().Regex).To(Equal(routePolicy.Methods[i]))
				Expect(route.GetRoute().GetWeightedClusters().TotalWeight).To(Equal(&wrappers.UInt32Value{Value: uint32(totalClusterWeight)}))
				Expect(len(route.GetRoute().GetWeightedClusters().GetClusters())).To(Equal(len(weightedClusters)))
				for j, cluster := range route.GetRoute().GetWeightedClusters().GetClusters() {
					Expect(cluster.Name).To(Equal(string(weightedClusters[j].ClusterName) + envoy.LocalClusterSuffix))
					Expect(cluster.Weight).To(Equal(&wrappers.UInt32Value{Value: uint32(weightedClusters[j].Weight)}))
				}
			}
		})

		It("Appends another route", func() {

			routePolicy2 := endpoint.RoutePolicy{
				PathRegex: "/buy-a-book",
				Methods:   []string{"GET"},
			}
			routeWeightedClustersList = append(routeWeightedClustersList, endpoint.RoutePolicyWeightedClusters{RoutePolicy: routePolicy2, WeightedClusters: weightedClusters})
			httpMethodCount := 3 // 2 from previously added routes + 1 append
			rt := createRoutes(routeWeightedClustersList, true)
			Expect(len(rt)).To(Equal(httpMethodCount))
			Expect(rt[2].Match.GetSafeRegex().Regex).To(Equal(routePolicy2.PathRegex))
			Expect(rt[2].Match.GetHeaders()[0].GetSafeRegexMatch().Regex).To(Equal(routePolicy2.Methods[0]))
			Expect(rt[2].GetRoute().GetWeightedClusters().TotalWeight).To(Equal(&wrappers.UInt32Value{Value: uint32(totalClusterWeight)}))
			Expect(len(rt[2].GetRoute().GetWeightedClusters().GetClusters())).To(Equal(len(weightedClusters)))
			for j, cluster := range rt[2].GetRoute().GetWeightedClusters().GetClusters() {
				Expect(cluster.Name).To(Equal(string(weightedClusters[j].ClusterName) + envoy.LocalClusterSuffix))
				Expect(cluster.Weight).To(Equal(&wrappers.UInt32Value{Value: uint32(weightedClusters[j].Weight)}))
			}
		})
	})
})

var _ = Describe("Route Configuration", func() {
	Context("Testing creation of RouteConfiguration object", func() {
		It("Returns outbound route configuration", func() {

			weightedClusters := []endpoint.WeightedCluster{
				{ClusterName: endpoint.ClusterName("osm/bookstore-1"), Weight: 100},
				{ClusterName: endpoint.ClusterName("osm/bookstore-2"), Weight: 100},
			}
			totalClusterWeight := 0
			for _, cluster := range weightedClusters {
				totalClusterWeight += cluster.Weight
			}
			routePolicy := endpoint.RoutePolicy{
				PathRegex: "/books-bought",
				Methods:   []string{"GET"},
			}
			routePolicyWeightedClustersList := []endpoint.RoutePolicyWeightedClusters{
				{RoutePolicy: routePolicy, WeightedClusters: weightedClusters},
			}
			sourceDomainAggregatedData := map[string][]endpoint.RoutePolicyWeightedClusters{
				"bookstore.mesh": routePolicyWeightedClustersList,
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
			Expect(len(sourceRouteConfig.VirtualHosts[0].Routes[0].GetRoute().GetWeightedClusters().GetClusters())).To(Equal(len(weightedClusters)))
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
