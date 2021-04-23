package catalog

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/openservicemesh/osm/pkg/tests"
)

var _ = Describe("Test catalog proxy register/unregister", func() {
	mc := newFakeMeshCatalog()

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
			trafficSplits, serviceAccounts, routeGroups, trafficTargets := mc.ListSMIPolicies()

			Expect(trafficSplits[0].Spec.Service).To(Equal("bookstore-apex"))
			Expect(serviceAccounts[0].String()).To(Equal("default/bookstore"))
			Expect(routeGroups[0].Name).To(Equal("bookstore-service-routes"))
			Expect(trafficTargets[0].Name).To(Equal(tests.TrafficTargetName))

		})
	})
})
