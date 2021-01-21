package catalog

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/tests"
)

var _ = Describe("Test catalog proxy register/unregister", func() {
	mc := newFakeMeshCatalog()
	certCommonName := certificate.CommonName("foo")
	certSerialNumber := certificate.SerialNumber("123456")
	proxy := envoy.NewProxy(certCommonName, certSerialNumber, nil)

	Context("Test register/unregister proxies", func() {
		It("no proxies expected, connected or disconnected", func() {
			expectedProxies := mc.ListExpectedProxies()
			Expect(len(expectedProxies)).To(Equal(0))

			connectedProxies := mc.ListConnectedProxies()
			Expect(len(connectedProxies)).To(Equal(0))

			disconnectedProxies := mc.ListDisconnectedProxies()
			Expect(len(disconnectedProxies)).To(Equal(0))
		})

		It("expect one proxy to connect", func() {
			// mc.RegisterProxy(proxy)
			mc.ExpectProxy(certCommonName)

			expectedProxies := mc.ListExpectedProxies()
			Expect(len(expectedProxies)).To(Equal(1))

			connectedProxies := mc.ListConnectedProxies()
			Expect(len(connectedProxies)).To(Equal(0))

			disconnectedProxies := mc.ListDisconnectedProxies()
			Expect(len(disconnectedProxies)).To(Equal(0))

			_, ok := expectedProxies[certCommonName]
			Expect(ok).To(BeTrue())
		})

		It("one proxy connected to OSM", func() {
			mc.RegisterProxy(proxy)

			expectedProxies := mc.ListExpectedProxies()
			Expect(len(expectedProxies)).To(Equal(0))

			connectedProxies := mc.ListConnectedProxies()
			Expect(len(connectedProxies)).To(Equal(1))

			disconnectedProxies := mc.ListDisconnectedProxies()
			Expect(len(disconnectedProxies)).To(Equal(0))

			_, ok := connectedProxies[certCommonName]
			Expect(ok).To(BeTrue())
		})

		It("one proxy disconnected from OSM", func() {
			mc.UnregisterProxy(proxy)

			expectedProxies := mc.ListExpectedProxies()
			Expect(len(expectedProxies)).To(Equal(0))

			connectedProxies := mc.ListConnectedProxies()
			Expect(len(connectedProxies)).To(Equal(0))

			disconnectedProxies := mc.ListDisconnectedProxies()
			Expect(len(disconnectedProxies)).To(Equal(1))

			_, ok := disconnectedProxies[certCommonName]
			Expect(ok).To(BeTrue())
		})
	})

	Context("Test ListMonitoredNamespaces", func() {
		It("lists monitored namespaces", func() {
			actual := mc.ListMonitoredNamespaces()
			listExpectedNs := tests.GetUnique([]string{
				tests.BookstoreV1Service.Namespace,
				tests.BookbuyerService.Namespace,
				tests.BookwarehouseService.Namespace,
			})

			Expect(actual).To(Equal(listExpectedNs))
		})
	})

	Context("Test ListSMIPolicies", func() {
		It("lists available SMI Spec policies", func() {
			trafficSplits, weightedServices, serviceAccounts, routeGroups, trafficTargets := mc.ListSMIPolicies()

			Expect(trafficSplits[0].Spec.Service).To(Equal("bookstore-apex"))
			Expect(weightedServices[0]).To(Equal(service.WeightedService{
				Service:     tests.BookstoreV1Service,
				Weight:      tests.Weight90,
				RootService: "bookstore-apex"}))
			Expect(serviceAccounts[0].String()).To(Equal("default/bookstore"))
			Expect(routeGroups[0].Name).To(Equal("bookstore-service-routes"))
			Expect(trafficTargets[0].Name).To(Equal(tests.TrafficTargetName))

		})
	})
})
