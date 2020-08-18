package kubernetes

import (
	"fmt"

	//"strings"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/openservicemesh/osm/pkg/tests"
)

var _ = Describe("Hostnames for a kubernetes service", func() {
	Context("Testing GethostnamesForService", func() {
		contains := func(hostnames []string, expected string) bool {
			for _, entry := range hostnames {
				if entry == expected {
					return true
				}
			}
			return false
		}
		It("Returns a list of hostnames corresponding to the service", func() {
			selectors := map[string]string{
				tests.SelectorKey: tests.SelectorValue,
			}
			service := tests.NewServiceFixture(tests.BookbuyerServiceName, tests.Namespace, selectors)
			hostnames := GetHostnamesForService(service)
			Expect(len(hostnames)).To(Equal(10))
			Expect(contains(hostnames, tests.BookbuyerServiceName)).To(BeTrue())
			Expect(contains(hostnames, fmt.Sprintf("%s:%d", tests.BookbuyerServiceName, tests.ServicePort))).To(BeTrue())
			Expect(contains(hostnames, fmt.Sprintf("%s.%s", tests.BookbuyerServiceName, tests.Namespace))).To(BeTrue())
			Expect(contains(hostnames, fmt.Sprintf("%s.%s:%d", tests.BookbuyerServiceName, tests.Namespace, tests.ServicePort))).To(BeTrue())
			Expect(contains(hostnames, fmt.Sprintf("%s.%s.svc", tests.BookbuyerServiceName, tests.Namespace))).To(BeTrue())
			Expect(contains(hostnames, fmt.Sprintf("%s.%s.svc:%d", tests.BookbuyerServiceName, tests.Namespace, tests.ServicePort))).To(BeTrue())
			Expect(contains(hostnames, fmt.Sprintf("%s.%s.svc.cluster", tests.BookbuyerServiceName, tests.Namespace))).To(BeTrue())
			Expect(contains(hostnames, fmt.Sprintf("%s.%s.svc.cluster:%d", tests.BookbuyerServiceName, tests.Namespace, tests.ServicePort))).To(BeTrue())
			Expect(contains(hostnames, fmt.Sprintf("%s.%s.svc.cluster.local", tests.BookbuyerServiceName, tests.Namespace))).To(BeTrue())
			Expect(contains(hostnames, fmt.Sprintf("%s.%s.svc.cluster.local:%d", tests.BookbuyerServiceName, tests.Namespace, tests.ServicePort))).To(BeTrue())
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
			Expect(GetServiceFromHostname(hostname)).To(Equal(fmt.Sprintf("%s:%d", service, tests.ServicePort)))
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
