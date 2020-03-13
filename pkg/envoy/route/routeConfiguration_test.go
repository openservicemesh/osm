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
					{Name: "smc/bookstore-1", Weight: &wrappers.UInt32Value{Value: uint32(50)}},
					{Name: "smc/bookstore-2", Weight: &wrappers.UInt32Value{Value: uint32(50)}}},
				TotalWeight: &wrappers.UInt32Value{Value: uint32(100)},
			}

			srcWeightedClusters := route.WeightedCluster{
				Clusters: []*route.WeightedCluster_ClusterWeight{
					{Name: "smc/bookstore-1-local", Weight: &wrappers.UInt32Value{Value: uint32(50)}},
					{Name: "smc/bookstore-2-local", Weight: &wrappers.UInt32Value{Value: uint32(50)}}},
				TotalWeight: &wrappers.UInt32Value{Value: uint32(100)},
			}

			trafficPolicies := endpoint.TrafficTargetPolicies{
				PolicyName: "bookbuyer-bookstore",
				Destination: endpoint.TrafficResource{
					ServiceAccount: "bookstore-serviceaccount",
					Namespace:      "smc",
					Services: []endpoint.NamespacedService{
						endpoint.NamespacedService{Namespace: "smc", Service: "bookstore-1"},
						endpoint.NamespacedService{Namespace: "smc", Service: "bookstore-2"}},
					Clusters: []endpoint.WeightedCluster{
						{ClusterName: endpoint.ClusterName("smc/bookstore-1"), Weight: 50},
						{ClusterName: endpoint.ClusterName("smc/bookstore-2"), Weight: 50}},
				},
				Source: endpoint.TrafficResource{
					ServiceAccount: "bookbuyer-serviceaccount",
					Namespace:      "smc",
					Services: []endpoint.NamespacedService{
						endpoint.NamespacedService{Namespace: "smc", Service: "bookbuyer"}},
					Clusters: []endpoint.WeightedCluster{
						{ClusterName: endpoint.ClusterName("smc/bookstore-1"), Weight: 50},
						{ClusterName: endpoint.ClusterName("smc/bookstore-2"), Weight: 50}},
				},
				PolicyRoutePaths: []endpoint.RoutePaths{
					endpoint.RoutePaths{
						RoutePathRegex: "/counter",
						RouteMethods:   []string{"GET"},
					},
				},
			}

			//Validating the outbound clusters and routes
			sourceRouteConfig := NewOutboundRouteConfiguration()
			sourceRouteConfig = UpdateRouteConfiguration(trafficPolicies, sourceRouteConfig, true, false)
			Expect(sourceRouteConfig).NotTo(Equal(nil))
			Expect(sourceRouteConfig.Name).To(Equal(OutboundRouteConfig))
			Expect(len(sourceRouteConfig.VirtualHosts)).To(Equal(1))
			Expect(len(sourceRouteConfig.VirtualHosts[0].Routes)).To(Equal(1))
			Expect(sourceRouteConfig.VirtualHosts[0].Routes[0].Match.GetPrefix()).To(Equal("/counter"))
			Expect(sourceRouteConfig.VirtualHosts[0].Routes[0].GetRoute().GetWeightedClusters()).To(Equal(&destWeightedClusters))
			Expect(sourceRouteConfig.VirtualHosts[0].Cors.AllowMethods).To(Equal("GET"))
			//Validating the inbound clusters and routes
			destinationRouteConfig := NewInboundRouteConfiguration()
			destinationRouteConfig = UpdateRouteConfiguration(trafficPolicies, destinationRouteConfig, false, true)
			Expect(destinationRouteConfig).NotTo(Equal(nil))
			Expect(destinationRouteConfig.Name).To(Equal(InboundRouteConfig))
			Expect(len(destinationRouteConfig.VirtualHosts)).To(Equal(1))
			Expect(len(destinationRouteConfig.VirtualHosts[0].Routes)).To(Equal(1))
			Expect(destinationRouteConfig.VirtualHosts[0].Routes[0].Match.GetPrefix()).To(Equal("/counter"))
			Expect(destinationRouteConfig.VirtualHosts[0].Routes[0].GetRoute().GetWeightedClusters()).To(Equal(&srcWeightedClusters))
			Expect(destinationRouteConfig.VirtualHosts[0].Cors.AllowMethods).To(Equal("GET"))
		})
	})
})

var _ = Describe("Cors Allowed Methods", func() {
	Context("Testing updateAllowedMethods", func() {
		It("Returns a unique list of allowed methods", func() {

			allowedMethods := []string{"GET", "POST", "PUT", "POST", "GET", "GET"}
			allowedMethods = updateAllowedMethods(allowedMethods)

			expectedAllowedMethods := []string{"GET", "POST", "PUT"}
			Expect(allowedMethods).To(Equal(expectedAllowedMethods))

		})

		It("Returns a wildcard allowed method (*)", func() {
			allowedMethods := []string{"GET", "POST", "PUT", "POST", "GET", "GET", "*"}
			allowedMethods = updateAllowedMethods(allowedMethods)

			expectedAllowedMethods := []string{"*"}
			Expect(allowedMethods).To(Equal(expectedAllowedMethods))
		})
	})
})

var _ = Describe("Weighted clusters", func() {
	Context("Testing getnWeightedClusters", func() {
		It("validated the creation of weighted clusters", func() {

			weightedClusters := []endpoint.WeightedCluster{
				{ClusterName: endpoint.ClusterName("smc/bookstore-1"), Weight: 30},
				{ClusterName: endpoint.ClusterName("smc/bookstore-2"), Weight: 70},
			}

			routeWeightedClusters := getWeightedCluster(weightedClusters, true)
			Expect(routeWeightedClusters.TotalWeight).To(Equal(&wrappers.UInt32Value{Value: uint32(100)}))
			Expect(len(routeWeightedClusters.GetClusters())).To(Equal(2))
			Expect(routeWeightedClusters.GetClusters()[0].Name).To(Equal("smc/bookstore-1-local"))
			Expect(routeWeightedClusters.GetClusters()[0].Weight).To(Equal(&wrappers.UInt32Value{Value: uint32(30)}))
			Expect(routeWeightedClusters.GetClusters()[1].Name).To(Equal("smc/bookstore-2-local"))
			Expect(routeWeightedClusters.GetClusters()[1].Weight).To(Equal(&wrappers.UInt32Value{Value: uint32(70)}))

		})
	})
})

var _ = Describe("Route Action weighted clusters", func() {
	Context("Testing updateRouteActionWeightedClusters", func() {
		It("Returns the route action with newly added clusters", func() {

			weightedClusters := []endpoint.WeightedCluster{
				{ClusterName: endpoint.ClusterName("smc/bookstore-1"), Weight: 100},
			}

			newWeightedClusters := []endpoint.WeightedCluster{
				{ClusterName: endpoint.ClusterName("smc/bookstore-2"), Weight: 100},
			}

			route1 := createRoute("counter", weightedClusters, false)
			rt := []*route.Route{&route1}
			updatedAction := updateRouteActionWeightedClusters(*rt[0].GetRoute().GetWeightedClusters(), newWeightedClusters, false)
			Expect(updatedAction.Route.GetWeightedClusters().TotalWeight).To(Equal(&wrappers.UInt32Value{Value: uint32(200)}))
			Expect(len(updatedAction.Route.GetWeightedClusters().GetClusters())).To(Equal(2))
			Expect(updatedAction.Route.GetWeightedClusters().GetClusters()[0].Name).To(Equal("smc/bookstore-1"))
			Expect(updatedAction.Route.GetWeightedClusters().GetClusters()[0].Weight).To(Equal(&wrappers.UInt32Value{Value: uint32(100)}))
			Expect(updatedAction.Route.GetWeightedClusters().GetClusters()[1].Name).To(Equal("smc/bookstore-2"))
			Expect(updatedAction.Route.GetWeightedClusters().GetClusters()[1].Weight).To(Equal(&wrappers.UInt32Value{Value: uint32(100)}))

		})
	})
})
