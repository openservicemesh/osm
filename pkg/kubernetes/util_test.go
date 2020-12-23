package kubernetes

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/openservicemesh/osm/pkg/tests"
)

var _ = Describe("Hostnames for a kubernetes service", func() {
	Context("Testing GethostnamesForService", func() {
		It("Should correctly return a list of hostnames corresponding to a service in the same namespace", func() {
			selectors := map[string]string{
				tests.SelectorKey: tests.SelectorValue,
			}
			service := tests.NewServiceFixture(tests.BookbuyerServiceName, tests.Namespace, selectors)
			hostnames := GetHostnamesForService(service, true)
			Expect(len(hostnames)).To(Equal(10))
			Expect(tests.BookbuyerServiceName).To(BeElementOf(hostnames))
			Expect(fmt.Sprintf("%s:%d", tests.BookbuyerServiceName, tests.ServicePort)).To(BeElementOf(hostnames))
			Expect(fmt.Sprintf("%s.%s", tests.BookbuyerServiceName, tests.Namespace)).To(BeElementOf(hostnames))
			Expect(fmt.Sprintf("%s.%s:%d", tests.BookbuyerServiceName, tests.Namespace, tests.ServicePort)).To(BeElementOf(hostnames))
			Expect(fmt.Sprintf("%s.%s.svc", tests.BookbuyerServiceName, tests.Namespace)).To(BeElementOf(hostnames))
			Expect(fmt.Sprintf("%s.%s.svc:%d", tests.BookbuyerServiceName, tests.Namespace, tests.ServicePort)).To(BeElementOf(hostnames))
			Expect(fmt.Sprintf("%s.%s.svc.cluster", tests.BookbuyerServiceName, tests.Namespace)).To(BeElementOf(hostnames))
			Expect(fmt.Sprintf("%s.%s.svc.cluster:%d", tests.BookbuyerServiceName, tests.Namespace, tests.ServicePort)).To(BeElementOf(hostnames))
			Expect(fmt.Sprintf("%s.%s.svc.cluster.local", tests.BookbuyerServiceName, tests.Namespace)).To(BeElementOf(hostnames))
			Expect(fmt.Sprintf("%s.%s.svc.cluster.local:%d", tests.BookbuyerServiceName, tests.Namespace, tests.ServicePort)).To(BeElementOf(hostnames))
		})

		It("Should correctly return a list of hostnames corresponding to a service not in the same namespace", func() {
			selectors := map[string]string{
				tests.SelectorKey: tests.SelectorValue,
			}
			service := tests.NewServiceFixture(tests.BookbuyerServiceName, tests.Namespace, selectors)
			hostnames := GetHostnamesForService(service, false)
			Expect(len(hostnames)).To(Equal(8))
			Expect(fmt.Sprintf("%s.%s", tests.BookbuyerServiceName, tests.Namespace)).To(BeElementOf(hostnames))
			Expect(fmt.Sprintf("%s.%s:%d", tests.BookbuyerServiceName, tests.Namespace, tests.ServicePort)).To(BeElementOf(hostnames))
			Expect(fmt.Sprintf("%s.%s.svc", tests.BookbuyerServiceName, tests.Namespace)).To(BeElementOf(hostnames))
			Expect(fmt.Sprintf("%s.%s.svc:%d", tests.BookbuyerServiceName, tests.Namespace, tests.ServicePort)).To(BeElementOf(hostnames))
			Expect(fmt.Sprintf("%s.%s.svc.cluster", tests.BookbuyerServiceName, tests.Namespace)).To(BeElementOf(hostnames))
			Expect(fmt.Sprintf("%s.%s.svc.cluster:%d", tests.BookbuyerServiceName, tests.Namespace, tests.ServicePort)).To(BeElementOf(hostnames))
			Expect(fmt.Sprintf("%s.%s.svc.cluster.local", tests.BookbuyerServiceName, tests.Namespace)).To(BeElementOf(hostnames))
			Expect(fmt.Sprintf("%s.%s.svc.cluster.local:%d", tests.BookbuyerServiceName, tests.Namespace, tests.ServicePort)).To(BeElementOf(hostnames))
		})
	})

	Context("Testing GetServiceFromHostname", func() {
		service := "test-service"
		It("Returns the service name from its hostname", func() {
			hostname := service
			Expect(GetServiceFromHostname(hostname)).To(Equal(service))
		})
		It("Returns the service name from its hostname", func() {
			hostname := fmt.Sprintf("%s:%d", service, tests.ServicePort)
			Expect(GetServiceFromHostname(hostname)).To(Equal(service))
		})
		It("Returns the service name from its hostname", func() {
			hostname := fmt.Sprintf("%s.namespace", service)
			Expect(GetServiceFromHostname(hostname)).To(Equal(service))
		})
		It("Returns the service name from its hostname", func() {
			hostname := fmt.Sprintf("%s.namespace.svc", service)
			Expect(GetServiceFromHostname(hostname)).To(Equal(service))
		})
		It("Returns the service name from its hostname", func() {
			hostname := fmt.Sprintf("%s.namespace.svc.cluster", service)
			Expect(GetServiceFromHostname(hostname)).To(Equal(service))
		})
	})
})
