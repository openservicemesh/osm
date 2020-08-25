package catalog

import (
	"fmt"
	"strings"

	"github.com/google/go-cmp/cmp"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/smi"
	"github.com/openservicemesh/osm/pkg/tests"
	"github.com/openservicemesh/osm/pkg/trafficpolicy"
	"github.com/openservicemesh/osm/pkg/utils"
)

var _ = Describe("Catalog tests", func() {
	mc := newFakeMeshCatalog()

	Context("Test ListTrafficPolicies", func() {
		It("lists traffic policies", func() {
			actual, err := mc.ListTrafficPolicies(tests.BookstoreService)
			Expect(err).ToNot(HaveOccurred())

			expected := []trafficpolicy.TrafficTarget{tests.TrafficPolicy}
			Expect(actual).To(Equal(expected))
		})
	})

	Context("Test getTrafficPoliciesForService", func() {
		It("should return all the traffic policies associated with a service", func() {
			allTrafficPolicies, err := getTrafficPoliciesForService(mc, tests.RoutePolicyMap, tests.BookbuyerService)
			Expect(err).ToNot(HaveOccurred())

			expected := []trafficpolicy.TrafficTarget{
				{
					Name:        utils.GetTrafficTargetName(tests.TrafficTargetName, tests.BookbuyerService, tests.BookstoreService),
					Destination: tests.BookstoreService,
					Source:      tests.BookbuyerService,
					HTTPRoutes: []trafficpolicy.HTTPRoute{
						{
							PathRegex: tests.BookstoreBuyPath,
							Methods:   []string{"GET"},
							Headers: map[string]string{
								"user-agent": tests.HTTPUserAgent,
							},
						},
					},
				},
				{
					Name:        utils.GetTrafficTargetName(tests.TrafficTargetName, tests.BookbuyerService, tests.BookstoreApexService),
					Destination: tests.BookstoreApexService,
					Source:      tests.BookbuyerService,
					HTTPRoutes: []trafficpolicy.HTTPRoute{
						{
							PathRegex: tests.BookstoreBuyPath,
							Methods:   []string{"GET"},
							Headers: map[string]string{
								"user-agent": tests.HTTPUserAgent,
							},
						},
					},
				},
			}

			Expect(allTrafficPolicies).To(Equal(expected))
		})
	})

	Context("Test getHTTPPathsPerRoute", func() {
		mc := MeshCatalog{meshSpec: smi.NewFakeMeshSpecClient()}
		It("constructs HTTP paths per route", func() {
			actual, err := mc.getHTTPPathsPerRoute()
			Expect(err).ToNot(HaveOccurred())

			specKey := mc.getTrafficSpecName("HTTPRouteGroup", tests.Namespace, tests.RouteGroupName)
			expected := map[trafficpolicy.TrafficSpecName]map[trafficpolicy.TrafficSpecMatchName]trafficpolicy.HTTPRoute{
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
			expectedSourceTrafficResource := utils.K8sSvcToMeshSvc(source)
			destination := tests.NewServiceFixture(tests.BookstoreServiceName, tests.Namespace, selectors)
			expectedDestinationTrafficResource := utils.K8sSvcToMeshSvc(destination)

			expectedHostHeaders := map[string]string{"user-agent": tests.HTTPUserAgent}
			expectedRoute := trafficpolicy.HTTPRoute{
				PathRegex: constants.RegexMatchAll,
				Methods:   []string{constants.WildcardHTTPMethod},
				Headers:   expectedHostHeaders,
			}

			trafficTarget := mc.buildAllowPolicyForSourceToDest(source, destination)
			Expect(cmp.Equal(trafficTarget.Source, expectedSourceTrafficResource)).To(BeTrue())
			Expect(cmp.Equal(trafficTarget.Destination, expectedDestinationTrafficResource)).To(BeTrue())
			Expect(cmp.Equal(trafficTarget.HTTPRoutes[0].PathRegex, expectedRoute.PathRegex)).To(BeTrue())
			Expect(cmp.Equal(trafficTarget.HTTPRoutes[0].Methods, expectedRoute.Methods)).To(BeTrue())
		})
	})

	Context("Test ListAllowedOutboundServices()", func() {
		It("returns the list of server names the given service is allowed to communicate with", func() {
			// mc := NewFakemc(testclient.NewSimpleClientset())
			actualList, err := mc.ListAllowedOutboundServices(tests.BookbuyerService)
			Expect(err).ToNot(HaveOccurred())
			expectedList := []service.MeshService{tests.BookstoreService, tests.BookstoreApexService}
			Expect(len(actualList)).To(Equal(len(expectedList)))
			Expect(actualList[0]).To(BeElementOf(expectedList))
			Expect(actualList[1]).To(BeElementOf(expectedList))
		})
	})

	Context("Test GetWeightedClusterForService()", func() {
		It("returns weighted clusters for a given service", func() {
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
			actual := mc.buildAllowAllTrafficPolicies(tests.BookstoreService)
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

	Context("Test listTrafficTargetPermutations", func() {
		It("lists n*m list of traffic targets for the given services", func() {
			destList := []service.MeshService{tests.BookstoreService, tests.BookstoreApexService}
			srcList := []service.MeshService{tests.BookbuyerService, tests.BookwarehouseService}
			trafficTargets := listTrafficTargetPermutations(tests.TrafficTargetName, srcList, destList)

			Expect(len(trafficTargets)).To(Equal(len(destList) * len(srcList)))
		})
	})

	Context("Test hashSrcDstService", func() {
		It("Should correctly hash a source and destination service to its key", func() {
			src := service.MeshService{
				Namespace: "src-ns",
				Name:      "source",
			}
			dst := service.MeshService{
				Namespace: "dst-ns",
				Name:      "destination",
			}

			srcDstServiceHash := hashSrcDstService(src, dst)
			Expect(srcDstServiceHash).To(Equal("src-ns/source:dst-ns/destination"))
		})
	})

	Context("Test getTrafficTargetFromSrcDstHash", func() {

		It("Should correctly return a traffic target from its hash key", func() {
			src := service.MeshService{
				Namespace: "src-ns",
				Name:      "source",
			}
			dst := service.MeshService{
				Namespace: "dst-ns",
				Name:      "destination",
			}
			srcDstServiceHash := "src-ns/source:dst-ns/destination"

			targetName := "test"
			httpRoutes := []trafficpolicy.HTTPRoute{
				{
					PathRegex: tests.BookstoreBuyPath,
					Methods:   []string{"GET"},
					Headers: map[string]string{
						"user-agent": tests.HTTPUserAgent,
					},
				},
			}

			trafficTarget := getTrafficTargetFromSrcDstHash(srcDstServiceHash, targetName, httpRoutes)

			expectedTrafficTarget := trafficpolicy.TrafficTarget{
				Source:      src,
				Destination: dst,
				Name:        targetName,
				HTTPRoutes:  httpRoutes,
			}
			Expect(trafficTarget).To(Equal(expectedTrafficTarget))
		})
	})
})
