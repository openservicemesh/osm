package catalog

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/go-cmp/cmp"
	testclient "k8s.io/client-go/kubernetes/fake"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/certificate/providers/tresor"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/endpoint"
	"github.com/openservicemesh/osm/pkg/endpoint/providers/kube"
	"github.com/openservicemesh/osm/pkg/ingress"
	"github.com/openservicemesh/osm/pkg/namespace"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/smi"
	"github.com/openservicemesh/osm/pkg/tests"
	"github.com/openservicemesh/osm/pkg/trafficpolicy"
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

	namespaceController := namespace.NewFakeNamespaceController([]string{osmNamespace})

	meshCatalog := NewMeshCatalog(namespaceController, kubeClient, smi.NewFakeMeshSpecClient(), certManager, ingress.NewFakeIngressMonitor(), make(<-chan struct{}), cfg, endpointProviders...)

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
				Name:        tests.TrafficTargetName,
				Destination: tests.BookstoreService,
				Source:      tests.BookbuyerService,
				Route: trafficpolicy.Route{PathRegex: tests.BookstoreBuyPath, Methods: []string{"GET"}, Headers: map[string]string{
					"user-agent": tests.HTTPUserAgent,
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
				specKey: {
					trafficpolicy.TrafficSpecMatchName(tests.BuyBooksMatchName): {
						PathRegex: tests.BookstoreBuyPath,
						Methods:   []string{"GET"},
						Headers: map[string]string{
							"user-agent": tests.HTTPUserAgent,
						}},
					trafficpolicy.TrafficSpecMatchName(tests.SellBooksMatchName): {
						PathRegex: tests.BookstoreSellPath,
						Methods:   []string{"GET"},
					},
					trafficpolicy.TrafficSpecMatchName(tests.WildcardWithHeadersMatchName): {
						PathRegex: ".*",
						Methods:   []string{"*"},
						Headers: map[string]string{
							"user-agent": tests.HTTPUserAgent,
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
			expectedList := []service.MeshService{tests.BookbuyerService}
			Expect(actualList).To(Equal(expectedList))
		})
	})

	Context("Testing buildAllowPolicyForSourceToDest", func() {
		It("Returns a trafficpolicy.TrafficTarget object to build an allow policy from source to destination service ", func() {
			selectors := map[string]string{
				tests.SelectorKey: tests.SelectorValue,
			}
			source := tests.NewServiceFixture(tests.BookbuyerServiceName, tests.Namespace, selectors)
			expectedSourceTrafficResource := service.MeshService{
				Namespace: source.Namespace,
				Name:      source.Name,
			}
			destination := tests.NewServiceFixture(tests.BookstoreServiceName, tests.Namespace, selectors)
			expectedDestinationTrafficResource := service.MeshService{
				Namespace: destination.Namespace,
				Name:      destination.Name,
			}

			expectedHostHeaders := map[string]string{"user-agent": tests.HTTPUserAgent}
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
			expectedList := []service.MeshService{tests.BookstoreService}
			Expect(actualList).To(Equal(expectedList))

		})
	})

	Context("Test GetWeightedClusterForService()", func() {
		It("returns weighted clusters for a given service", func() {
			mc := newFakeMeshCatalog()
			weightedCluster, err := mc.GetWeightedClusterForService(tests.BookstoreService)
			Expect(err).ToNot(HaveOccurred())
			expected := service.WeightedCluster{
				ClusterName: "default/bookstore",
				Weight:      100,
			}
			Expect(weightedCluster).To(Equal(expected))

		})
	})

	Context("Test catalog functions", func() {
		It("getServiceHostnames list of service hostnames", func() {
			mc := newFakeMeshCatalog()
			actual, err := mc.getServiceHostnames(tests.BookstoreService)
			Expect(err).ToNot(HaveOccurred())
			expected := []string{
				"bookstore",
				"bookstore.default",
				"bookstore.default.svc",
				"bookstore.default.svc.cluster",
				"bookstore.default.svc.cluster.local",
				"bookstore:8888",
				"bookstore.default:8888",
				"bookstore.default.svc:8888",
				"bookstore.default.svc.cluster:8888",
				"bookstore.default.svc.cluster.local:8888",
			}
			Expect(actual).To(Equal(expected))
		})

		It("hostnamesTostr returns CSV of hostnames", func() {
			actual := hostnamesTostr([]string{"foo", "bar", "baz"})
			expected := "foo,bar,baz"
			Expect(actual).To(Equal(expected))
		})

		It("getDefaultWeightedClusterForService returns correct WeightedCluster struct", func() {
			actual := getDefaultWeightedClusterForService(tests.BookstoreService)
			expected := service.WeightedCluster{
				ClusterName: "default/bookstore",
				Weight:      100,
			}
			Expect(actual).To(Equal(expected))
		})
	})

	Context("Test GetHostnamesForService()", func() {
		It("lists available SMI Spec policies in SMI mode", func() {
			mc := newFakeMeshCatalog()
			actual, err := mc.GetHostnamesForService(tests.BookstoreService)
			Expect(err).ToNot(HaveOccurred())
			expected := strings.Join(
				[]string{
					"bookstore-apex",
					"bookstore-apex.default",
					"bookstore-apex.default.svc",
					"bookstore-apex.default.svc.cluster",
					"bookstore-apex.default.svc.cluster.local",
					"bookstore-apex:8888",
					"bookstore-apex.default:8888",
					"bookstore-apex.default.svc:8888",
					"bookstore-apex.default.svc.cluster:8888",
					"bookstore-apex.default.svc.cluster.local:8888",
				},
				",")
			Expect(actual).To(Equal(expected))
		})
	})

	Context("Test buildAllowAllTrafficPolicies", func() {
		It("lists traffic targets for the given service", func() {
			mc := newFakeMeshCatalog()
			actual, err := mc.buildAllowAllTrafficPolicies(tests.BookstoreService)
			Expect(err).ToNot(HaveOccurred())
			var actualTargetNames []string
			for _, target := range actual {
				actualTargetNames = append(actualTargetNames, target.Name)
			}

			expected := []string{
				"default/bookstore->default/bookbuyer",
				"default/bookstore->default/bookstore-apex",
				"default/bookbuyer->default/bookstore",
				"default/bookbuyer->default/bookstore-apex",
				"default/bookstore-apex->default/bookstore",
				"default/bookstore-apex->default/bookbuyer",
			}
			Expect(actualTargetNames).To(Equal(expected))
		})
	})

})
