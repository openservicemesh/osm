package catalog

import (
	"fmt"

	testclient "k8s.io/client-go/kubernetes/fake"

	set "github.com/deckarep/golang-set"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/open-service-mesh/osm/pkg/certificate/providers/tresor"
	"github.com/open-service-mesh/osm/pkg/endpoint"
	"github.com/open-service-mesh/osm/pkg/ingress"
	"github.com/open-service-mesh/osm/pkg/providers/kube"
	"github.com/open-service-mesh/osm/pkg/smi"
	"github.com/open-service-mesh/osm/pkg/tests"
)

var _ = Describe("Catalog tests", func() {
	endpointProviders := []endpoint.Provider{kube.NewFakeProvider()}
	kubeClient := testclient.NewSimpleClientset()
	meshCatalog := NewMeshCatalog(kubeClient, smi.NewFakeMeshSpecClient(), tresor.NewFakeCertManager(), ingress.NewFakeIngressMonitor(), make(<-chan struct{}), endpointProviders...)

	Context("Test ListTrafficPolicies", func() {
		It("lists traffic policies", func() {
			actual, err := meshCatalog.ListTrafficPolicies(tests.BookstoreService)
			Expect(err).ToNot(HaveOccurred())

			expected := []endpoint.TrafficPolicy{tests.TrafficPolicy}
			Expect(actual).To(Equal(expected))
		})
	})

	Context("Test getActiveServices", func() {
		It("lists active services", func() {
			actual := meshCatalog.getActiveServices(set.NewSet(tests.BookstoreService))
			expected := set.NewSet(endpoint.NamespacedService{
				Namespace: "default",
				Service:   "bookstore",
			})
			Expect(actual.Equal(expected)).To(Equal(true))
		})
	})

	Context("Test getTrafficPolicyPerRoute", func() {
		It("lists traffic policies", func() {
			allTrafficPolicies, err := getTrafficPolicyPerRoute(meshCatalog, tests.RoutePolicyMap, tests.BookstoreService)
			Expect(err).ToNot(HaveOccurred())

			expected := []endpoint.TrafficPolicy{{
				PolicyName: tests.TrafficTargetName,
				Destination: endpoint.TrafficPolicyResource{
					ServiceAccount: tests.BookstoreServiceAccountName,
					Namespace:      tests.Namespace,
					Services:       set.NewSet(tests.BookstoreService),
				},
				Source: endpoint.TrafficPolicyResource{
					ServiceAccount: tests.BookbuyerServiceAccountName,
					Namespace:      tests.Namespace,
					Services:       set.NewSet(tests.BookbuyerService),
				},
				RoutePolicies: []endpoint.RoutePolicy{{PathRegex: "", Methods: nil}},
			}}

			Expect(allTrafficPolicies).To(Equal(expected))
		})
	})

	Context("Test listServicesForServiceAccount", func() {
		mc := MeshCatalog{
			serviceAccountToServicesCache: map[endpoint.NamespacedServiceAccount][]endpoint.NamespacedService{
				tests.BookstoreServiceAccount: {tests.BookstoreService},
			},
		}
		It("lists services for service account", func() {
			actual, err := mc.listServicesForServiceAccount(tests.BookstoreServiceAccount)
			Expect(err).ToNot(HaveOccurred())
			expected := set.NewSet(endpoint.NamespacedService{
				Namespace: tests.Namespace,
				Service:   tests.BookstoreServiceName,
			})
			Expect(actual.Equal(expected)).To(Equal(true))
		})
	})

	Context("Test getHTTPPathsPerRoute", func() {
		mc := MeshCatalog{meshSpec: smi.NewFakeMeshSpecClient()}
		It("constructs HTTP paths per route", func() {
			actual, err := mc.getHTTPPathsPerRoute()
			Expect(err).ToNot(HaveOccurred())

			key := fmt.Sprintf("HTTPRouteGroup/%s/%s/%s", tests.Namespace, tests.RouteGroupName, tests.MatchName)
			expected := map[string]endpoint.RoutePolicy{
				key: {
					PathRegex: tests.BookstoreBuyPath,
					Methods:   []string{"GET"},
				},
			}
			Expect(actual).To(Equal(expected))
		})
	})
})
