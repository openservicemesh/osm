package catalog

import (
	"fmt"

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
	meshCatalog := NewMeshCatalog(smi.NewFakeMeshSpecClient(), tresor.NewFakeCertManager(), ingress.NewFakeIngressMonitor(), make(<-chan struct{}), endpointProviders...)

	Context("Testing UniqueLists", func() {
		It("Returns unique list of services", func() {

			services := []endpoint.NamespacedService{
				{Namespace: "osm", Service: "booktore-1"},
				{Namespace: "osm", Service: "booktore-1"},
				{Namespace: "osm", Service: "booktore-2"},
				{Namespace: "osm", Service: "booktore-3"},
				{Namespace: "osm", Service: "booktore-2"},
			}

			actual := uniqueServices(services)
			expected := []endpoint.NamespacedService{
				{Namespace: "osm", Service: "booktore-1"},
				{Namespace: "osm", Service: "booktore-2"},
				{Namespace: "osm", Service: "booktore-3"},
			}
			Expect(actual).To(Equal(expected))
		})
	})

	Context("Testing servicesToString", func() {
		It("Returns string list", func() {

			services := []endpoint.NamespacedService{
				{Namespace: "osm", Service: "bookstore-1"},
				{Namespace: "osm", Service: "bookstore-2"},
			}

			actual := servicesToString(services)
			expected := []string{
				"osm/bookstore-1",
				"osm/bookstore-2",
			}
			Expect(actual).To(Equal(expected))
		})
	})

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
			actual := meshCatalog.getActiveServices([]endpoint.NamespacedService{tests.BookstoreService})
			expected := []endpoint.NamespacedService{{
				Namespace: "default",
				Service:   "bookstore",
			}}
			Expect(actual).To(Equal(expected))
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
					Services:       []endpoint.NamespacedService{tests.BookstoreService},
				},
				Source: endpoint.TrafficPolicyResource{
					ServiceAccount: tests.BookbuyerServiceAccountName,
					Namespace:      tests.Namespace,
					Services:       []endpoint.NamespacedService{tests.BookbuyerService},
				},
				RoutePolicies: []endpoint.RoutePolicy{{PathRegex: "", Methods: nil}},
			}}

			Expect(allTrafficPolicies).To(Equal(expected))
		})
	})

	Context("Test listServicesForServiceAccount", func() {
		mc := MeshCatalog{
			serviceAccountsCache: map[endpoint.NamespacedServiceAccount][]endpoint.NamespacedService{
				tests.BookstoreServiceAccount: {tests.BookstoreService},
			},
		}
		It("lists services for service account", func() {
			actual, err := mc.listServicesForServiceAccount(tests.BookstoreServiceAccount)
			Expect(err).ToNot(HaveOccurred())
			expected := []endpoint.NamespacedService{{
				Namespace: tests.Namespace,
				Service:   tests.BookstoreServiceName,
			}}
			Expect(actual).To(Equal(expected))
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
