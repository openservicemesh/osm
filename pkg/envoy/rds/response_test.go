package rds

import (
	"time"

	set "github.com/deckarep/golang-set"
	testclient "k8s.io/client-go/kubernetes/fake"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/open-service-mesh/osm/pkg/catalog"
	"github.com/open-service-mesh/osm/pkg/certificate"
	"github.com/open-service-mesh/osm/pkg/certificate/providers/tresor"
	"github.com/open-service-mesh/osm/pkg/configurator"
	"github.com/open-service-mesh/osm/pkg/constants"
	"github.com/open-service-mesh/osm/pkg/endpoint"
	"github.com/open-service-mesh/osm/pkg/endpoint/providers/kube"
	"github.com/open-service-mesh/osm/pkg/ingress"
	"github.com/open-service-mesh/osm/pkg/service"
	"github.com/open-service-mesh/osm/pkg/smi"
	"github.com/open-service-mesh/osm/pkg/tests"
	"github.com/open-service-mesh/osm/pkg/trafficpolicy"
)

const (
	testHeaderKey1 = "test-header-1"
	testHeaderKey2 = "test-header-2"
)

var _ = Describe("Construct RoutePolicyWeightedClusters object", func() {
	Context("Testing the creating of a RoutePolicyWeightedClusters object", func() {
		It("Returns RoutePolicyWeightedClusters", func() {

			weightedCluster := service.WeightedCluster{ClusterName: service.ClusterName("osm/bookstore-1"), Weight: 100}
			routePolicy := trafficpolicy.Route{
				PathRegex: "/books-bought",
				Methods:   []string{"GET"},
			}

			routePolicyWeightedClusters := createRoutePolicyWeightedClusters(routePolicy, weightedCluster)
			Expect(routePolicyWeightedClusters).NotTo(Equal(nil))
			Expect(routePolicyWeightedClusters.Route.PathRegex).To(Equal("/books-bought"))
			Expect(routePolicyWeightedClusters.Route.Methods).To(Equal([]string{"GET"}))
			Expect(routePolicyWeightedClusters.WeightedClusters.Cardinality()).To(Equal(1))
			routePolicyWeightedClustersSlice := routePolicyWeightedClusters.WeightedClusters.ToSlice()
			Expect(string(routePolicyWeightedClustersSlice[0].(service.WeightedCluster).ClusterName)).To(Equal("osm/bookstore-1"))
			Expect(routePolicyWeightedClustersSlice[0].(service.WeightedCluster).Weight).To(Equal(100))
		})
	})
})

var _ = Describe("AggregateRoutesByDomain", func() {
	domainRoutesMap := make(map[string]map[string]trafficpolicy.RouteWeightedClusters)
	weightedClustersMap := set.NewSet()
	Context("Building a map of routes by domain", func() {
		It("Returns a new aggregated map of domain and routes", func() {

			weightedCluster := service.WeightedCluster{ClusterName: service.ClusterName("osm/bookstore-1"), Weight: 100}
			routePolicies := []trafficpolicy.Route{
				{PathRegex: "/books-bought", Methods: []string{"GET"}},
				{PathRegex: "/buy-a-book", Methods: []string{"GET"}},
			}
			weightedClustersMap.Add(weightedCluster)

			for _, routePolicy := range routePolicies {
				aggregateRoutesByDomain(domainRoutesMap, routePolicy, weightedCluster, "bookstore.mesh")
			}
			Expect(domainRoutesMap).NotTo(Equal(nil))
			Expect(len(domainRoutesMap)).To(Equal(1))
			Expect(len(domainRoutesMap["bookstore.mesh"])).To(Equal(2))

			for _, routePolicy := range routePolicies {
				_, routePolicyExists := domainRoutesMap["bookstore.mesh"][routePolicy.PathRegex]
				Expect(routePolicyExists).To(Equal(true))
			}
			for path := range domainRoutesMap["bookstore.mesh"] {
				Expect(domainRoutesMap["bookstore.mesh"][path].WeightedClusters.Cardinality()).To(Equal(1))
				Expect(domainRoutesMap["bookstore.mesh"][path].WeightedClusters.Equal(weightedClustersMap)).To(Equal(true))
			}
		})
	})

	Context("Adding a route to existing domain in the map", func() {
		It("Returns the map of with a new route on the domain", func() {

			weightedCluster := service.WeightedCluster{ClusterName: service.ClusterName("osm/bookstore-1"), Weight: constants.ClusterWeightAcceptAll}
			routePolicy := trafficpolicy.Route{
				PathRegex: "/update-books-bought",
				Methods:   []string{"GET"},
				Headers: map[string]string{
					testHeaderKey1: "This is a test header 1",
				},
			}
			weightedClustersMap.Add(weightedCluster)

			aggregateRoutesByDomain(domainRoutesMap, routePolicy, weightedCluster, "bookstore.mesh")
			Expect(domainRoutesMap).NotTo(Equal(nil))
			Expect(len(domainRoutesMap)).To(Equal(1))
			Expect(len(domainRoutesMap["bookstore.mesh"])).To(Equal(3))
			Expect(domainRoutesMap["bookstore.mesh"][routePolicy.PathRegex].Route).To(Equal(routePolicy))
			Expect(domainRoutesMap["bookstore.mesh"][routePolicy.PathRegex].WeightedClusters.Cardinality()).To(Equal(1))
			Expect(domainRoutesMap["bookstore.mesh"][routePolicy.PathRegex].Route).To(Equal(trafficpolicy.Route{PathRegex: "/update-books-bought", Methods: []string{"GET"}, Headers: map[string]string{testHeaderKey1: "This is a test header 1"}}))
			Expect(domainRoutesMap["bookstore.mesh"][routePolicy.PathRegex].WeightedClusters.Equal(weightedClustersMap)).To(Equal(true))
		})
	})

	Context("Adding a cluster to an existing route to existing domain in the map", func() {
		It("Returns the map of with a new weighted cluster on a route in the domain", func() {

			weightedCluster := service.WeightedCluster{ClusterName: service.ClusterName("osm/bookstore-2"), Weight: constants.ClusterWeightAcceptAll}
			routePolicy := trafficpolicy.Route{
				PathRegex: "/update-books-bought",
				Methods:   []string{"GET"},
				Headers: map[string]string{
					testHeaderKey2: "This is a test header 2",
				},
			}
			weightedClustersMap.Add(weightedCluster)

			aggregateRoutesByDomain(domainRoutesMap, routePolicy, weightedCluster, "bookstore.mesh")
			Expect(domainRoutesMap).NotTo(Equal(nil))
			Expect(len(domainRoutesMap)).To(Equal(1))
			Expect(len(domainRoutesMap["bookstore.mesh"])).To(Equal(3))
			Expect(domainRoutesMap["bookstore.mesh"][routePolicy.PathRegex].WeightedClusters.Cardinality()).To(Equal(2))
			Expect(domainRoutesMap["bookstore.mesh"][routePolicy.PathRegex].Route).To(Equal(trafficpolicy.Route{PathRegex: "/update-books-bought", Methods: []string{"GET", "GET"}, Headers: map[string]string{testHeaderKey1: "This is a test header 1", testHeaderKey2: "This is a test header 2"}}))
			Expect(domainRoutesMap["bookstore.mesh"][routePolicy.PathRegex].WeightedClusters.Equal(weightedClustersMap)).To(Equal(true))
		})
	})
})

var _ = Describe("RDS Response", func() {
	endpointProviders := []endpoint.Provider{kube.NewFakeProvider()}
	kubeClient := testclient.NewSimpleClientset()
	cache := make(map[certificate.CommonName]certificate.Certificater)
	certManager := tresor.NewFakeCertManager(&cache, 1*time.Hour)

	stop := make(<-chan struct{})
	osmNamespace := "-test-osm-namespace-"
	osmConfigMapName := "-test-osm-config-map-"
	cfg := configurator.NewConfigurator(kubeClient, stop, osmNamespace, osmConfigMapName)

	meshCatalog := catalog.NewMeshCatalog(kubeClient, smi.NewFakeMeshSpecClient(), certManager, ingress.NewFakeIngressMonitor(), make(<-chan struct{}), cfg, endpointProviders...)

	Context("Test GetDomainsForService", func() {
		It("returns domain for service from traffic split", func() {

			actual, err := meshCatalog.GetDomainForService(tests.BookstoreService, tests.HTTPRouteGroup.Matches[0].Headers)
			Expect(err).ToNot(HaveOccurred())
			Expect(actual).To(Equal(tests.Domain))
		})

		It("returns domain for service from traffic spec http header host", func() {

			actual, err := meshCatalog.GetDomainForService(tests.BookbuyerService, tests.HTTPRouteGroup.Matches[0].Headers)
			Expect(err).ToNot(HaveOccurred())
			Expect(actual).To(Equal(tests.Domain))
		})

		It("no domain found for service", func() {

			_, err := meshCatalog.GetDomainForService(tests.BookbuyerService, tests.HTTPRouteGroup.Matches[1].Headers)
			Expect(err).To(HaveOccurred())
		})
	})

	Context("GetWeightedClusterForService", func() {
		It("returns weighted cluster for service from traffic split", func() {

			actual, err := meshCatalog.GetWeightedClusterForService(tests.BookstoreService)
			Expect(err).ToNot(HaveOccurred())

			expected := service.WeightedCluster{
				ClusterName: service.ClusterName(tests.WeightedService.NamespacedService.String()),
				Weight:      tests.WeightedService.Weight,
			}
			Expect(actual).To(Equal(expected))
		})

		It("returns weighted cluster for service with default weight of 100", func() {

			actual, err := meshCatalog.GetWeightedClusterForService(tests.BookbuyerService)
			Expect(err).ToNot(HaveOccurred())

			expected := service.WeightedCluster{
				ClusterName: service.ClusterName(tests.BookbuyerService.String()),
				Weight:      tests.Weight,
			}
			Expect(actual).To(Equal(expected))
		})
	})
})
