package route

import (
	"github.com/golang/protobuf/ptypes/wrappers"
	"github.com/open-service-mesh/osm/pkg/endpoint"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
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

			routeWeightedClusters := getWeightedCluster(weightedClusters, true)
			Expect(routeWeightedClusters.TotalWeight).To(Equal(&wrappers.UInt32Value{Value: uint32(100)}))
			Expect(len(routeWeightedClusters.GetClusters())).To(Equal(2))
			Expect(routeWeightedClusters.GetClusters()[0].Name).To(Equal("osm/bookstore-1-local"))
			Expect(routeWeightedClusters.GetClusters()[0].Weight).To(Equal(&wrappers.UInt32Value{Value: uint32(30)}))
			Expect(routeWeightedClusters.GetClusters()[1].Name).To(Equal("osm/bookstore-2-local"))
			Expect(routeWeightedClusters.GetClusters()[1].Weight).To(Equal(&wrappers.UInt32Value{Value: uint32(70)}))

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

		It("Adds a new route", func() {

			routePolicy := endpoint.RoutePolicy{
				PathRegex: "/books-bought",
				Methods:   []string{"GET", "POST"},
			}
			routeWeightedClustersList = append(routeWeightedClustersList, endpoint.RoutePolicyWeightedClusters{RoutePolicy: routePolicy, WeightedClusters: weightedClusters})

			rt := createRoutes(routeWeightedClustersList, false)
			Expect(len(rt)).To(Equal(1))
			Expect(rt[0].Match.GetSafeRegex().Regex).To(Equal("/books-bought"))
			Expect(rt[0].Match.GetHeaders()[0].GetExactMatch()).To(Equal("GET"))
			Expect(rt[0].Match.GetHeaders()[1].GetExactMatch()).To(Equal("POST"))
			Expect(rt[0].GetRoute().GetWeightedClusters().TotalWeight).To(Equal(&wrappers.UInt32Value{Value: uint32(200)}))
			Expect(len(rt[0].GetRoute().GetWeightedClusters().GetClusters())).To(Equal(2))
			Expect(rt[0].GetRoute().GetWeightedClusters().GetClusters()[0].Name).To(Equal("osm/bookstore-1"))
			Expect(rt[0].GetRoute().GetWeightedClusters().GetClusters()[0].Weight).To(Equal(&wrappers.UInt32Value{Value: uint32(100)}))
			Expect(rt[0].GetRoute().GetWeightedClusters().GetClusters()[1].Name).To(Equal("osm/bookstore-2"))
			Expect(rt[0].GetRoute().GetWeightedClusters().GetClusters()[1].Weight).To(Equal(&wrappers.UInt32Value{Value: uint32(100)}))

		})

		It("Appends another route", func() {

			routePolicy2 := endpoint.RoutePolicy{
				PathRegex: "/buy-a-book",
				Methods:   []string{"GET"},
			}
			routeWeightedClustersList = append(routeWeightedClustersList, endpoint.RoutePolicyWeightedClusters{RoutePolicy: routePolicy2, WeightedClusters: weightedClusters})

			rt := createRoutes(routeWeightedClustersList, false)
			Expect(len(rt)).To(Equal(2))
			Expect(rt[1].Match.GetSafeRegex().Regex).To(Equal("/buy-a-book"))
			Expect(rt[1].Match.GetHeaders()[0].GetExactMatch()).To(Equal("GET"))
			Expect(rt[1].GetRoute().GetWeightedClusters().TotalWeight).To(Equal(&wrappers.UInt32Value{Value: uint32(200)}))
			Expect(len(rt[1].GetRoute().GetWeightedClusters().GetClusters())).To(Equal(2))
			Expect(rt[1].GetRoute().GetWeightedClusters().GetClusters()[0].Name).To(Equal("osm/bookstore-1"))
			Expect(rt[1].GetRoute().GetWeightedClusters().GetClusters()[0].Weight).To(Equal(&wrappers.UInt32Value{Value: uint32(100)}))
			Expect(rt[1].GetRoute().GetWeightedClusters().GetClusters()[1].Name).To(Equal("osm/bookstore-2"))
			Expect(rt[1].GetRoute().GetWeightedClusters().GetClusters()[1].Weight).To(Equal(&wrappers.UInt32Value{Value: uint32(100)}))

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
			Expect(len(sourceRouteConfig.VirtualHosts)).To(Equal(1))
			Expect(len(sourceRouteConfig.VirtualHosts[0].Routes)).To(Equal(1))
			Expect(sourceRouteConfig.VirtualHosts[0].Routes[0].Match.GetSafeRegex().Regex).To(Equal("/books-bought"))
			Expect(sourceRouteConfig.VirtualHosts[0].Routes[0].Match.GetHeaders()[0].GetExactMatch()).To(Equal("GET"))
			Expect(len(sourceRouteConfig.VirtualHosts[0].Routes[0].GetRoute().GetWeightedClusters().GetClusters())).To(Equal(2))
		})
	})
})
