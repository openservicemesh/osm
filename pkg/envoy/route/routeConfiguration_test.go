package route

import (
	route "github.com/envoyproxy/go-control-plane/envoy/api/v2/route"
	"github.com/golang/protobuf/ptypes/wrappers"
	"github.com/open-service-mesh/osm/pkg/endpoint"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Route Configuration", func() {
	Context("Testing RouteConfiguration", func() {
		It("Returns route configuration", func() {
			destWeightedClusters := route.WeightedCluster{
				Clusters: []*route.WeightedCluster_ClusterWeight{
					{Name: "osm/bookstore-1", Weight: &wrappers.UInt32Value{Value: uint32(50)}},
					{Name: "osm/bookstore-2", Weight: &wrappers.UInt32Value{Value: uint32(50)}}},
				TotalWeight: &wrappers.UInt32Value{Value: uint32(100)},
			}

			srcWeightedClusters := route.WeightedCluster{
				Clusters: []*route.WeightedCluster_ClusterWeight{
					{Name: "osm/bookstore-1-local", Weight: &wrappers.UInt32Value{Value: uint32(50)}},
					{Name: "osm/bookstore-2-local", Weight: &wrappers.UInt32Value{Value: uint32(50)}}},
				TotalWeight: &wrappers.UInt32Value{Value: uint32(100)},
			}

			trafficPolicies := endpoint.TrafficTargetPolicies{
				PolicyName: "bookbuyer-bookstore",
				Destination: endpoint.TrafficResource{
					ServiceAccount: "bookstore-serviceaccount",
					Namespace:      "osm",
					Services: []endpoint.NamespacedService{
						endpoint.NamespacedService{Namespace: "osm", Service: "bookstore-1"},
						endpoint.NamespacedService{Namespace: "osm", Service: "bookstore-2"}},
					Clusters: []endpoint.WeightedCluster{
						{ClusterName: endpoint.ClusterName("osm/bookstore-1"), Weight: 50},
						{ClusterName: endpoint.ClusterName("osm/bookstore-2"), Weight: 50}},
				},
				Source: endpoint.TrafficResource{
					ServiceAccount: "bookbuyer-serviceaccount",
					Namespace:      "osm",
					Services: []endpoint.NamespacedService{
						endpoint.NamespacedService{Namespace: "osm", Service: "bookbuyer"}},
					Clusters: []endpoint.WeightedCluster{
						{ClusterName: endpoint.ClusterName("osm/bookstore-1"), Weight: 50},
						{ClusterName: endpoint.ClusterName("osm/bookstore-2"), Weight: 50}},
				},
				PolicyRoutePaths: []endpoint.RoutePaths{
					endpoint.RoutePaths{
						RoutePathRegex: "/books-bought",
						RouteMethods:   []string{"GET"},
					},
				},
				Domains: []string{"bookstore.mesh"},
			}

			//Validating the outbound clusters and routes
			sourceRouteConfig := NewRouteConfiguration(OutboundRouteConfig)
			sourceRouteConfig = UpdateRouteConfiguration(trafficPolicies, sourceRouteConfig, true, false)
			Expect(sourceRouteConfig).NotTo(Equal(nil))
			Expect(sourceRouteConfig.Name).To(Equal(OutboundRouteConfig))
			Expect(len(sourceRouteConfig.VirtualHosts)).To(Equal(1))
			Expect(len(sourceRouteConfig.VirtualHosts[0].Routes)).To(Equal(1))
			Expect(sourceRouteConfig.VirtualHosts[0].Routes[0].Match.GetSafeRegex().Regex).To(Equal("/books-bought"))
			Expect(sourceRouteConfig.VirtualHosts[0].Routes[0].Match.GetHeaders()[0].GetExactMatch()).To(Equal("GET"))
			Expect(sourceRouteConfig.VirtualHosts[0].Routes[0].GetRoute().GetWeightedClusters()).To(Equal(&destWeightedClusters))
			Expect(sourceRouteConfig.VirtualHosts[0].Domains).To(Equal([]string{"bookstore.mesh"}))
			//Validating the inbound clusters and routes
			destinationRouteConfig := NewRouteConfiguration(InboundRouteConfig)
			destinationRouteConfig = UpdateRouteConfiguration(trafficPolicies, destinationRouteConfig, false, true)
			Expect(destinationRouteConfig).NotTo(Equal(nil))
			Expect(destinationRouteConfig.Name).To(Equal(InboundRouteConfig))
			Expect(len(destinationRouteConfig.VirtualHosts)).To(Equal(1))
			Expect(len(destinationRouteConfig.VirtualHosts[0].Routes)).To(Equal(1))
			Expect(destinationRouteConfig.VirtualHosts[0].Routes[0].Match.GetSafeRegex().Regex).To(Equal("/books-bought"))
			Expect(sourceRouteConfig.VirtualHosts[0].Routes[0].Match.GetHeaders()[0].GetExactMatch()).To(Equal("GET"))
			Expect(destinationRouteConfig.VirtualHosts[0].Routes[0].GetRoute().GetWeightedClusters()).To(Equal(&srcWeightedClusters))
			Expect(destinationRouteConfig.VirtualHosts[0].Domains).To(Equal([]string{"bookstore.mesh"}))
		})
	})

	Context("Testing RouteConfiguration with multiple domains", func() {
		It("Returns route configuration with two virtual hosts", func() {
			trafficPolicy1 := endpoint.TrafficTargetPolicies{
				PolicyName: "bookbuyer-bookstore",
				Destination: endpoint.TrafficResource{
					ServiceAccount: "bookstore-serviceaccount",
					Namespace:      "osm",
					Services: []endpoint.NamespacedService{
						endpoint.NamespacedService{Namespace: "osm", Service: "bookstore-1"},
						endpoint.NamespacedService{Namespace: "osm", Service: "bookstore-2"}},
					Clusters: []endpoint.WeightedCluster{
						{ClusterName: endpoint.ClusterName("osm/bookstore-1"), Weight: 50},
						{ClusterName: endpoint.ClusterName("osm/bookstore-2"), Weight: 50}},
				},
				Source: endpoint.TrafficResource{
					ServiceAccount: "bookbuyer-serviceaccount",
					Namespace:      "osm",
					Services: []endpoint.NamespacedService{
						endpoint.NamespacedService{Namespace: "osm", Service: "bookbuyer"}},
					Clusters: []endpoint.WeightedCluster{
						{ClusterName: endpoint.ClusterName("osm/bookstore-1"), Weight: 50},
						{ClusterName: endpoint.ClusterName("osm/bookstore-2"), Weight: 50}},
				},
				PolicyRoutePaths: []endpoint.RoutePaths{
					endpoint.RoutePaths{
						RoutePathRegex: "/books-bought",
						RouteMethods:   []string{"GET"},
					},
				},
				Domains: []string{"bookstore.mesh"},
			}

			trafficPolicy2 := endpoint.TrafficTargetPolicies{
				PolicyName: "bookbuyer-bookstore",
				Destination: endpoint.TrafficResource{
					ServiceAccount: "bookstore-serviceaccount",
					Namespace:      "osm",
					Services: []endpoint.NamespacedService{
						endpoint.NamespacedService{Namespace: "osm", Service: "bookstore-1"},
						endpoint.NamespacedService{Namespace: "osm", Service: "bookstore-2"}},
					Clusters: []endpoint.WeightedCluster{
						{ClusterName: endpoint.ClusterName("osm/bookstore-1"), Weight: 50},
						{ClusterName: endpoint.ClusterName("osm/bookstore-2"), Weight: 50}},
				},
				Source: endpoint.TrafficResource{
					ServiceAccount: "bookbuyer-serviceaccount",
					Namespace:      "osm",
					Services: []endpoint.NamespacedService{
						endpoint.NamespacedService{Namespace: "osm", Service: "bookbuyer"}},
					Clusters: []endpoint.WeightedCluster{
						{ClusterName: endpoint.ClusterName("osm/bookstore-1"), Weight: 50},
						{ClusterName: endpoint.ClusterName("osm/bookstore-2"), Weight: 50}},
				},
				PolicyRoutePaths: []endpoint.RoutePaths{
					endpoint.RoutePaths{
						RoutePathRegex: "/books-bought",
						RouteMethods:   []string{"GET"},
					},
				},
				Domains: []string{"bookinventory.mesh"},
			}

			//Validating the outbound clusters and routes
			sourceRouteConfig := NewRouteConfiguration(OutboundRouteConfig)
			sourceRouteConfig = UpdateRouteConfiguration(trafficPolicy1, sourceRouteConfig, true, false)
			sourceRouteConfig = UpdateRouteConfiguration(trafficPolicy2, sourceRouteConfig, true, false)
			Expect(sourceRouteConfig).NotTo(Equal(nil))
			Expect(sourceRouteConfig.Name).To(Equal(OutboundRouteConfig))
			Expect(len(sourceRouteConfig.VirtualHosts)).To(Equal(2))
			Expect(len(sourceRouteConfig.VirtualHosts[0].Routes)).To(Equal(1))
			Expect(sourceRouteConfig.VirtualHosts[0].Domains).To(Equal([]string{"bookstore.mesh"}))
			Expect(len(sourceRouteConfig.VirtualHosts[1].Routes)).To(Equal(1))
			Expect(sourceRouteConfig.VirtualHosts[1].Domains).To(Equal([]string{"bookinventory.mesh"}))
			//Validating the inbound clusters and routes
			destinationRouteConfig := NewRouteConfiguration(InboundRouteConfig)
			destinationRouteConfig = UpdateRouteConfiguration(trafficPolicy1, destinationRouteConfig, false, true)
			destinationRouteConfig = UpdateRouteConfiguration(trafficPolicy2, destinationRouteConfig, false, true)
			Expect(destinationRouteConfig).NotTo(Equal(nil))
			Expect(destinationRouteConfig.Name).To(Equal(InboundRouteConfig))
			Expect(len(destinationRouteConfig.VirtualHosts)).To(Equal(2))
			Expect(len(destinationRouteConfig.VirtualHosts[0].Routes)).To(Equal(1))
			Expect(destinationRouteConfig.VirtualHosts[0].Domains).To(Equal([]string{"bookstore.mesh"}))
			Expect(len(destinationRouteConfig.VirtualHosts[1].Routes)).To(Equal(1))
			Expect(destinationRouteConfig.VirtualHosts[1].Domains).To(Equal([]string{"bookinventory.mesh"}))
		})
	})
})

var _ = Describe("Cors Allowed Methods", func() {
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
	Context("Testing getnWeightedClusters", func() {
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

var _ = Describe("Route Action weighted clusters", func() {
	Context("Testing updateRouteActionWeightedClusters", func() {
		It("Returns the route action with newly added clusters", func() {

			weightedClusters := []endpoint.WeightedCluster{
				{ClusterName: endpoint.ClusterName("osm/bookstore-1"), Weight: 100},
			}

			newWeightedClusters := []endpoint.WeightedCluster{
				{ClusterName: endpoint.ClusterName("osm/bookstore-2"), Weight: 100},
			}

			routePath := endpoint.RoutePaths{
				RoutePathRegex: "books-bought",
				RouteMethods:   []string{"GET", "POST"},
			}
			route1 := createRoute(&routePath, weightedClusters, false)
			rt := []*route.Route{&route1}
			updatedAction := updateRouteActionWeightedClusters(*rt[0].GetRoute().GetWeightedClusters(), newWeightedClusters, false)
			Expect(updatedAction.Route.GetWeightedClusters().TotalWeight).To(Equal(&wrappers.UInt32Value{Value: uint32(200)}))
			Expect(len(updatedAction.Route.GetWeightedClusters().GetClusters())).To(Equal(2))
			Expect(updatedAction.Route.GetWeightedClusters().GetClusters()[0].Name).To(Equal("osm/bookstore-1"))
			Expect(updatedAction.Route.GetWeightedClusters().GetClusters()[0].Weight).To(Equal(&wrappers.UInt32Value{Value: uint32(100)}))
			Expect(updatedAction.Route.GetWeightedClusters().GetClusters()[1].Name).To(Equal("osm/bookstore-2"))
			Expect(updatedAction.Route.GetWeightedClusters().GetClusters()[1].Weight).To(Equal(&wrappers.UInt32Value{Value: uint32(100)}))

		})
	})
})
