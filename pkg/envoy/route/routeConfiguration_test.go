package route

import (
	"github.com/deislabs/smc/pkg/endpoint"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Route Configuration", func() {
	Context("Testing RouteConfiguration", func() {
		It("Returns route configuration", func() {
			trafficPolicies := endpoint.TrafficTargetPolicies{
				PolicyName: "bookbuyer-bookstore",
				Destination: endpoint.TrafficResource{
					ServiceAccount: "bookstore-serviceaccount",
					Namespace:      "smc",
					Services:       []endpoint.ServiceName{endpoint.ServiceName("smc/bookstore-1"), endpoint.ServiceName("smc/bookstore-2")},
					Clusters:       []endpoint.ServiceName{endpoint.ServiceName("smc/bookstore-1"), endpoint.ServiceName("smc/bookstore-2")},
				},
				Source: endpoint.TrafficResource{
					ServiceAccount: "bookbuyer-serviceaccount",
					Namespace:      "smc",
					Services:       []endpoint.ServiceName{endpoint.ServiceName("smc/bookbuyer")},
					Clusters:       []endpoint.ServiceName{endpoint.ServiceName("smc/bookstore.mesh")},
				},
				PolicyRoutePaths: []endpoint.RoutePaths{
					endpoint.RoutePaths{
						RoutePathRegex: "/counter",
						RouteMethods:   []string{"GET"},
					},
				},
			}

			routeConfig := NewRouteConfiguration(trafficPolicies)
			Expect(routeConfig).NotTo(Equal(nil))
			Expect(len(routeConfig)).To(Equal(2))
			//Validating the destination clusters and routes
			Expect(routeConfig[0].Name).To(Equal(DestinationRouteConfig))
			Expect(len(routeConfig[0].VirtualHosts[0].Routes)).To(Equal(2))
			Expect(routeConfig[0].VirtualHosts[0].Cors.AllowMethods).To(Equal("GET"))
			Expect(routeConfig[0].VirtualHosts[0].Routes[0].Match.GetPrefix()).To(Equal("/counter"))
			Expect(routeConfig[0].VirtualHosts[0].Routes[0].GetRoute().GetCluster()).To(Equal("smc/bookstore-1"))
			Expect(routeConfig[0].VirtualHosts[0].Routes[1].GetRoute().GetCluster()).To(Equal("smc/bookstore-2"))
			//Validating the source clusters and routes
			Expect(routeConfig[1].Name).To(Equal(SourceRouteConfig))
			Expect(len(routeConfig[1].VirtualHosts)).To(Equal(1))
			Expect(len(routeConfig[1].VirtualHosts[0].Routes)).To(Equal(1))
			Expect(routeConfig[1].VirtualHosts[0].Cors.AllowMethods).To(Equal("GET"))
			Expect(routeConfig[1].VirtualHosts[0].Routes[0].Match.GetPrefix()).To(Equal("/counter"))

		})
	})
})
