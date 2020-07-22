package catalog

import (
	"fmt"
	"time"

	"github.com/google/go-cmp/cmp"
	testclient "k8s.io/client-go/kubernetes/fake"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

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
	"github.com/open-service-mesh/osm/pkg/namespace"
)

var _ = Describe("Catalog tests", func() {
	endpointProviders := []endpoint.Provider{kube.NewFakeProvider()}
	kubeClient := testclient.NewSimpleClientset()
	cache := make(map[certificate.CommonName]certificate.Certificater)
	certManager := tresor.NewFakeCertManager(&cache, 1*time.Hour)

	stop := make(<-chan struct{})
	osmNamespace := "-test-osm-namespace-"
	osmConfigMapName := "-test-osm-config-map-"
	cfg := configurator.NewConfigurator(kubeClient, stop, osmNamespace, osmConfigMapName)

	namespaceCtrlr := namespace.NewFakeNamespaceController([]string{osmNamespace})

	meshCatalog := NewMeshCatalog(namespaceCtrlr, kubeClient, smi.NewFakeMeshSpecClient(), certManager, ingress.NewFakeIngressMonitor(), make(<-chan struct{}), cfg, endpointProviders...)

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
					Namespace: tests.Namespace,
					Service:   tests.BookstoreService,
				},
				Source: trafficpolicy.TrafficResource{
					Namespace: tests.Namespace,
					Service:   tests.BookbuyerService,
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

	Context("Test ListAllowedInboundServices()", func() {
		It("returns the list of server names allowed to communicate with the hosted service", func() {
			mc := NewFakeMeshCatalog(testclient.NewSimpleClientset())
			actualList, err := mc.ListAllowedInboundServices(tests.BookstoreService)
			Expect(err).ToNot(HaveOccurred())
			expectedList := []service.NamespacedService{tests.BookbuyerService}
			Expect(actualList).To(Equal(expectedList))
		})
	})

	Context("Testing buildAllowPolicyForSourceToDest", func() {
		It("Returns a trafficpolicy.TrafficTarget object to build an allow policy from source to destination service ", func() {
			selectors := map[string]string{
				tests.SelectorKey: tests.SelectorValue,
			}
			source := tests.NewServiceFixture(tests.BookbuyerServiceName, tests.Namespace, selectors)
			expectedSourceTrafficResource := trafficpolicy.TrafficResource{
				Namespace: source.Namespace,
				Service: service.NamespacedService{
					Namespace: source.Namespace,
					Service:   source.Name,
				},
			}
			destination := tests.NewServiceFixture(tests.BookstoreServiceName, tests.Namespace, selectors)
			expectedDestinationTrafficResource := trafficpolicy.TrafficResource{
				Namespace: destination.Namespace,
				Service: service.NamespacedService{
					Namespace: destination.Namespace,
					Service:   destination.Name,
				},
			}

			expectedHostHeaders := map[string]string{HostHeaderKey: tests.BookstoreServiceName}
			expectedRoute := trafficpolicy.Route{
				PathRegex: constants.RegexMatchAll,
				Methods:   []string{constants.WildcardHTTPMethod},
				Headers:   expectedHostHeaders,
			}

			trafficTarget := meshCatalog.buildAllowPolicyForSourceToDest(source, destination)
			Expect(cmp.Equal(trafficTarget.Source, expectedSourceTrafficResource)).To(BeTrue())
			Expect(cmp.Equal(trafficTarget.Destination, expectedDestinationTrafficResource)).To(BeTrue())
			Expect(cmp.Equal(trafficTarget.Route.PathRegex, expectedRoute.PathRegex)).To(BeTrue())
			Expect(cmp.Equal(trafficTarget.Route.Methods, expectedRoute.Methods)).To(BeTrue())
		})
	})

	Context("Test ListAllowedOutboundServices()", func() {
		It("returns the list of server names the given service is allowed to communicate with", func() {
			mc := NewFakeMeshCatalog(testclient.NewSimpleClientset())
			actualList, err := mc.ListAllowedOutboundServices(tests.BookbuyerService)
			Expect(err).ToNot(HaveOccurred())
			expectedList := []service.NamespacedService{tests.BookstoreService}
			Expect(actualList).To(Equal(expectedList))

		})
	})
})
