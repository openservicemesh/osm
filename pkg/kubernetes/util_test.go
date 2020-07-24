package kubernetes

import (
	"fmt"
	//"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/open-service-mesh/osm/pkg/tests"
)

var _ = Describe("Domains for a kubernetes service", func() {
	Context("Testing GetDomainsForService", func() {
		contains := func(domains []string, expected string) bool {
			for _, entry := range domains {
				if entry == expected {
					return true
				}
			}
			return false
		}
		It("Returns a list of domains corresponding to the service", func() {
			selectors := map[string]string{
				tests.SelectorKey: tests.SelectorValue,
			}
			service := tests.NewServiceFixture(tests.BookbuyerServiceName, tests.Namespace, selectors)
			domains := GetDomainsForService(service)
			Expect(len(domains)).To(Equal(10))
			Expect(contains(domains, tests.BookbuyerServiceName)).To(BeTrue())
			Expect(contains(domains, fmt.Sprintf("%s:%d", tests.BookbuyerServiceName, tests.ServicePort))).To(BeTrue())
			Expect(contains(domains, fmt.Sprintf("%s.%s", tests.BookbuyerServiceName, tests.Namespace))).To(BeTrue())
			Expect(contains(domains, fmt.Sprintf("%s.%s:%d", tests.BookbuyerServiceName, tests.Namespace, tests.ServicePort))).To(BeTrue())
			Expect(contains(domains, fmt.Sprintf("%s.%s.svc", tests.BookbuyerServiceName, tests.Namespace))).To(BeTrue())
			Expect(contains(domains, fmt.Sprintf("%s.%s.svc:%d", tests.BookbuyerServiceName, tests.Namespace, tests.ServicePort))).To(BeTrue())
			Expect(contains(domains, fmt.Sprintf("%s.%s.svc.cluster", tests.BookbuyerServiceName, tests.Namespace))).To(BeTrue())
			Expect(contains(domains, fmt.Sprintf("%s.%s.svc.cluster:%d", tests.BookbuyerServiceName, tests.Namespace, tests.ServicePort))).To(BeTrue())
			Expect(contains(domains, fmt.Sprintf("%s.%s.svc.cluster.local", tests.BookbuyerServiceName, tests.Namespace))).To(BeTrue())
			Expect(contains(domains, fmt.Sprintf("%s.%s.svc.cluster.local:%d", tests.BookbuyerServiceName, tests.Namespace, tests.ServicePort))).To(BeTrue())
		})
	})
})
