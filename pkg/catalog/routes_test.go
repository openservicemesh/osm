package catalog

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/open-service-mesh/osm/pkg/certificate"
	"github.com/open-service-mesh/osm/pkg/certificate/providers/tresor"
	"github.com/open-service-mesh/osm/pkg/endpoint"
	"github.com/open-service-mesh/osm/pkg/ingress"
	"github.com/open-service-mesh/osm/pkg/providers/kube"
	"github.com/open-service-mesh/osm/pkg/service"
	"github.com/open-service-mesh/osm/pkg/smi"
	"github.com/open-service-mesh/osm/pkg/tests"
	"github.com/open-service-mesh/osm/pkg/trafficpolicy"
	testclient "k8s.io/client-go/kubernetes/fake"
)

var _ = Describe("Catalog tests", func() {
	endpointProviders := []endpoint.Provider{kube.NewFakeProvider()}
	kubeClient := testclient.NewSimpleClientset()
	cache := make(map[certificate.CommonName]certificate.Certificater)
	certManager := tresor.NewFakeCertManager(&cache, 1*time.Hour)
	meshCatalog := NewMeshCatalog(kubeClient, smi.NewFakeMeshSpecClient(), certManager, ingress.NewFakeIngressMonitor(), make(<-chan struct{}), endpointProviders...)

	Context("Test ListTrafficPolicies", func() {
		It("lists traffic policies", func() {
			actual, err := meshCatalog.ListTrafficPolicies(tests.BookstoreService)
			Expect(err).ToNot(HaveOccurred())

			expected := []trafficpolicy.TrafficTarget{tests.TrafficPolicy}
			Expect(actual).To(Equal(expected))
		})
	})

	Context("Test getTrafficPolicyPerRoute", func() {
		It("lists traffic policies", func() {
			allTrafficPolicies, err := getTrafficPolicyPerRoute(meshCatalog, tests.RoutePolicyMap, tests.BookstoreService)
			Expect(err).ToNot(HaveOccurred())

			expected := []trafficpolicy.TrafficTarget{{
				Name: tests.TrafficTargetName,
				Destination: trafficpolicy.TrafficResource{
					ServiceAccount: tests.BookstoreServiceAccountName,
					Namespace:      tests.Namespace,
					Service:        tests.BookstoreService,
				},
				Source: trafficpolicy.TrafficResource{
					ServiceAccount: tests.BookbuyerServiceAccountName,
					Namespace:      tests.Namespace,
					Service:        tests.BookbuyerService,
				},
				Route: trafficpolicy.Route{PathRegex: tests.BookstoreBuyPath, Methods: []string{"GET"}, Headers: map[string]string{
					"host": tests.Domain,
				}},
			}}

			Expect(allTrafficPolicies).To(Equal(expected))
		})
	})

	Context("Test getHTTPPathsPerRoute", func() {
		mc := MeshCatalog{meshSpec: smi.NewFakeMeshSpecClient()}
		It("constructs HTTP paths per route", func() {
			actual, err := mc.getHTTPPathsPerRoute()
			Expect(err).ToNot(HaveOccurred())

			specKey := mc.getTrafficSpecName("HTTPRouteGroup", tests.Namespace, tests.RouteGroupName)
			expected := map[trafficpolicy.TrafficSpecName]map[trafficpolicy.TrafficSpecMatchName]trafficpolicy.Route{
				specKey: map[trafficpolicy.TrafficSpecMatchName]trafficpolicy.Route{
					trafficpolicy.TrafficSpecMatchName(tests.BuyBooksMatchName): {
						PathRegex: tests.BookstoreBuyPath,
						Methods:   []string{"GET"},
						Headers: map[string]string{
							"host": tests.Domain,
						}},
					trafficpolicy.TrafficSpecMatchName(tests.SellBooksMatchName): {
						PathRegex: tests.BookstoreSellPath,
						Methods:   []string{"GET"},
					},
					trafficpolicy.TrafficSpecMatchName(tests.WildcardWithHeadersMatchName): {
						PathRegex: ".*",
						Methods:   []string{"*"},
						Headers: map[string]string{
							"host": tests.Domain,
						}}},
			}
			Expect(actual).To(Equal(expected))
		})
	})

	Context("Test getTrafficSpecName", func() {
		mc := MeshCatalog{meshSpec: smi.NewFakeMeshSpecClient()}
		It("returns the name of the TrafficSpec", func() {
			actual := mc.getTrafficSpecName("HTTPRouteGroup", tests.Namespace, tests.RouteGroupName)
			expected := trafficpolicy.TrafficSpecName(fmt.Sprintf("HTTPRouteGroup/%s/%s", tests.Namespace, tests.RouteGroupName))
			Expect(actual).To(Equal(expected))
		})
	})

	Context("Test ListAllowedPeerServices()", func() {
		It("returns the list of server names allowed to communicate with the hosted service", func() {
			mc := NewFakeMeshCatalog(testclient.NewSimpleClientset())
			actualList, err := mc.ListAllowedPeerServices(tests.BookstoreService)
			Expect(err).ToNot(HaveOccurred())
			expectedList := []service.NamespacedService{tests.BookbuyerService}
			Expect(actualList).To(Equal(expectedList))
		})
	})
})
