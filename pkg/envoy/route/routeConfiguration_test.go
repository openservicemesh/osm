package route

import (
	"github.com/open-service-mesh/osm/pkg/endpoint"
	route "github.com/envoyproxy/go-control-plane/envoy/api/v2/route"
	"github.com/golang/protobuf/ptypes/wrappers"

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
