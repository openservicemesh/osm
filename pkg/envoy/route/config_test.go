package route

import (
	"fmt"
	"strings"

	envoy_route "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"

	set "github.com/deckarep/golang-set"
	"github.com/golang/protobuf/ptypes/wrappers"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/tests"
	"github.com/openservicemesh/osm/pkg/trafficpolicy"
)

const (
	userAgentHeader = "user-agent"
)

var _ = Describe("VirtualHost cration", func() {
	Context("Testing createVirtualHostStub", func() {
		containsDomain := func(vhost *envoy_route.VirtualHost, domain string) bool {
			for _, entry := range vhost.Domains {
				if entry == domain {
					return true
				}
			}
			return false
		}
		It("Returns a VirtualHost object for a single domain", func() {
			prefix := "test"
			service := "test-service"
			domain := fmt.Sprintf("%s.namespace.svc.cluster.local", service)
			domains := set.NewSet(domain)

			vhost := createVirtualHostStub(prefix, service, domains)
			Expect(len(vhost.Domains)).To(Equal(1))
			Expect(vhost.Domains[0]).To(Equal(domain))
			Expect(vhost.Name).To(Equal(fmt.Sprintf("%s|%s", prefix, service)))
		})

		It("Returns a VirtualHost object for multiple comma seprated domains", func() {
			prefix := "test"
			service := "test-service"
			domain := fmt.Sprintf("%[1]s.namespace,%[1]s.namespace.svc,%[1]s.namespace.svc.cluster.local", service)
			expectedDomains := strings.Split(domain, ",")
			expectedDomainCount := len(expectedDomains)

			domainsSet := set.NewSet()
			for _, expectedDomain := range expectedDomains {
				domainsSet.Add(expectedDomain)
			}

			vhost := createVirtualHostStub(prefix, service, domainsSet)
			Expect(len(vhost.Domains)).To(Equal(expectedDomainCount))
			for _, entry := range expectedDomains {
				Expect(containsDomain(vhost, entry)).To(BeTrue())
			}
			Expect(vhost.Name).To(Equal(fmt.Sprintf("%s|%s", prefix, service)))
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
				clustersExpected.Add(string(envoy.GetLocalClusterNameForServiceCluster(cluster.ClusterName.String())))
				weightsExpected.Add(uint32(cluster.Weight))
			}

			totalClusterWeight := 0
			for clusterInterface := range weightedClusters.Iter() {
				cluster := clusterInterface.(service.WeightedCluster)
				totalClusterWeight += cluster.Weight
			}

			routeWeightedClusters := getWeightedCluster(weightedClusters, totalClusterWeight, InboundRoute)
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
			clustersExpected.Add(string(envoy.GetLocalClusterNameForServiceCluster(cluster.ClusterName.String())))
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

			routePolicy := trafficpolicy.HTTPRouteMatch{
				PathRegex: "/books-bought",
				Methods:   []string{"GET", "POST"},
			}

			routeWeightedClustersMap[routePolicy.PathRegex] = trafficpolicy.RouteWeightedClusters{HTTPRouteMatch: routePolicy, WeightedClusters: weightedClusters}
			rt := createRoutes(routeWeightedClustersMap, InboundRoute)
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

			routePolicy2 := trafficpolicy.HTTPRouteMatch{
				PathRegex: "/buy-a-book",
				Methods:   []string{"GET"},
			}
			routeWeightedClustersMap[routePolicy2.PathRegex] = trafficpolicy.RouteWeightedClusters{HTTPRouteMatch: routePolicy2, WeightedClusters: weightedClusters}

			httpMethodCount := 3 // 2 from previously added routes + 1 append

			rt := createRoutes(routeWeightedClustersMap, InboundRoute)

			Expect(len(rt)).To(Equal(httpMethodCount))
			var newRoute *envoy_route.Route
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

			domains := set.NewSet("bookstore.mesh")
			totalClusterWeight := 0
			for clusterInterface := range weightedClusters.Iter() {
				cluster := clusterInterface.(service.WeightedCluster)
				totalClusterWeight += cluster.Weight
			}
			routePolicy := trafficpolicy.HTTPRouteMatch{
				PathRegex: "/books-bought",
				Methods:   []string{"GET"},
				Headers: map[string]string{
					"user-agent": tests.HTTPUserAgent,
				},
			}

			sourceDomainRouteData := map[string]trafficpolicy.RouteWeightedClusters{
				routePolicy.PathRegex: {HTTPRouteMatch: routePolicy, WeightedClusters: weightedClusters, Hostnames: domains},
			}

			sourceDomainAggregatedData := map[string]map[string]trafficpolicy.RouteWeightedClusters{
				"bookstore": sourceDomainRouteData,
			}

			//Validating the outbound clusters and routes
			outboundRouteConfig := NewRouteConfigurationStub(OutboundRouteConfigName)
			UpdateRouteConfiguration(sourceDomainAggregatedData, outboundRouteConfig, OutboundRoute)
			Expect(outboundRouteConfig).NotTo(Equal(nil))
			Expect(outboundRouteConfig.Name).To(Equal(OutboundRouteConfigName))
			Expect(len(outboundRouteConfig.VirtualHosts)).To(Equal(len(sourceDomainAggregatedData)))
			Expect(len(outboundRouteConfig.VirtualHosts[0].Routes)).To(Equal(1))
			Expect(len(outboundRouteConfig.VirtualHosts[0].Routes[0].Match.Headers)).To(Equal(1))
			Expect(outboundRouteConfig.VirtualHosts[0].Routes[0].Match.GetSafeRegex().Regex).To(Equal(constants.RegexMatchAll))
			Expect(outboundRouteConfig.VirtualHosts[0].Routes[0].Match.GetHeaders()[0].GetSafeRegexMatch().Regex).To(Equal(constants.RegexMatchAll))
			Expect(len(outboundRouteConfig.VirtualHosts[0].Routes[0].GetRoute().GetWeightedClusters().GetClusters())).To(Equal(weightedClusters.Cardinality()))
			Expect(outboundRouteConfig.VirtualHosts[0].Routes[0].GetRoute().GetWeightedClusters().TotalWeight).To(Equal(&wrappers.UInt32Value{Value: uint32(totalClusterWeight)}))
		})

		It("Returns inbound route configuration", func() {

			weightedClusters := set.NewSet()
			weightedClusters.Add(service.WeightedCluster{ClusterName: service.ClusterName("osm/bookstore-1"), Weight: 100})

			domains := set.NewSet("bookstore.mesh")
			totalClusterWeight := 0
			for clusterInterface := range weightedClusters.Iter() {
				cluster := clusterInterface.(service.WeightedCluster)
				totalClusterWeight += cluster.Weight
			}
			routePolicy := trafficpolicy.HTTPRouteMatch{
				PathRegex: "/books-bought",
				Methods:   []string{"GET"},
				Headers: map[string]string{
					"user-agent": tests.HTTPUserAgent,
				},
			}

			destDomainRouteData := map[string]trafficpolicy.RouteWeightedClusters{
				routePolicy.PathRegex: {HTTPRouteMatch: routePolicy, WeightedClusters: weightedClusters, Hostnames: domains},
			}

			destDomainAggregatedData := map[string]map[string]trafficpolicy.RouteWeightedClusters{
				"bookstore": destDomainRouteData,
			}

			//Validating the inbound clusters and routes
			destRouteConfig := NewRouteConfigurationStub(InboundRouteConfigName)
			UpdateRouteConfiguration(destDomainAggregatedData, destRouteConfig, InboundRoute)
			Expect(destRouteConfig).NotTo(Equal(nil))
			Expect(destRouteConfig.Name).To(Equal(InboundRouteConfigName))
			Expect(len(destRouteConfig.VirtualHosts)).To(Equal(len(destDomainAggregatedData)))
			Expect(len(destRouteConfig.VirtualHosts[0].Routes)).To(Equal(1))
			Expect(len(destRouteConfig.VirtualHosts[0].Routes[0].Match.Headers)).To(Equal(2))
			Expect(destRouteConfig.VirtualHosts[0].Routes[0].Match.GetHeaders()[0].Name).To(Equal(MethodHeaderKey))
			Expect(destRouteConfig.VirtualHosts[0].Routes[0].Match.GetHeaders()[0].GetSafeRegexMatch().Regex).To(Equal(routePolicy.Methods[0]))
			Expect(destRouteConfig.VirtualHosts[0].Routes[0].Match.GetHeaders()[1].Name).To(Equal(userAgentHeader))
			Expect(destRouteConfig.VirtualHosts[0].Routes[0].Match.GetHeaders()[1].GetSafeRegexMatch().Regex).To(Equal(routePolicy.Headers[userAgentHeader]))
			Expect(len(destRouteConfig.VirtualHosts[0].Routes[0].GetRoute().GetWeightedClusters().GetClusters())).To(Equal(weightedClusters.Cardinality()))
			Expect(destRouteConfig.VirtualHosts[0].Routes[0].GetRoute().GetWeightedClusters().TotalWeight).To(Equal(&wrappers.UInt32Value{Value: uint32(totalClusterWeight)}))
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

var _ = Describe("Routes with headers", func() {
	Context("Testing getHeadersForRoute", func() {
		It("Returns a list of HeaderMatcher for a route", func() {
			routePolicy := trafficpolicy.HTTPRouteMatch{
				PathRegex: "/books-bought",
				Methods:   []string{"GET", "POST"},
				Headers: map[string]string{
					userAgentHeader: "This is a test header",
				},
			}
			headers := getHeadersForRoute(routePolicy.Methods[0], routePolicy.Headers)
			noOfHeaders := len(routePolicy.Headers) + 1 // an additional header for the methods
			Expect(len(headers)).To(Equal(noOfHeaders))
			Expect(headers[0].Name).To(Equal(MethodHeaderKey))
			Expect(headers[0].GetSafeRegexMatch().Regex).To(Equal(routePolicy.Methods[0]))
			Expect(headers[1].Name).To(Equal(userAgentHeader))
			Expect(headers[1].GetSafeRegexMatch().Regex).To(Equal(routePolicy.Headers[userAgentHeader]))
		})

		It("Returns only one HeaderMatcher for a route", func() {
			routePolicy := trafficpolicy.HTTPRouteMatch{
				PathRegex: "/books-bought",
				Methods:   []string{"GET", "POST"},
			}
			headers := getHeadersForRoute(routePolicy.Methods[1], routePolicy.Headers)
			noOfHeaders := len(routePolicy.Headers) + 1 // an additional header for the methods
			Expect(len(headers)).To(Equal(noOfHeaders))
			Expect(headers[0].Name).To(Equal(MethodHeaderKey))
			Expect(headers[0].GetSafeRegexMatch().Regex).To(Equal(routePolicy.Methods[1]))
		})

		It("Returns only one HeaderMatcher for a route ignoring the host", func() {
			routePolicy := trafficpolicy.HTTPRouteMatch{
				PathRegex: "/books-bought",
				Methods:   []string{"GET", "POST"},
				Headers: map[string]string{
					"user-agent": tests.HTTPUserAgent,
				},
			}
			headers := getHeadersForRoute(routePolicy.Methods[0], routePolicy.Headers)
			Expect(len(headers)).To(Equal(2))
			Expect(headers[0].Name).To(Equal(MethodHeaderKey))
			Expect(headers[0].GetSafeRegexMatch().Regex).To(Equal(routePolicy.Methods[0]))
		})
	})
})
