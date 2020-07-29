package endpoint

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/service"
)

var _ = Describe("Test NamespacedService methods", func() {
	Context("Testing GetCommonName", func() {
		It("should return DNS-1123 of the NamespacedService struct", func() {
			namespacedService := service.NamespacedService{
				Namespace: "namespace-here",
				Service:   "service-name-here",
			}
			actual := namespacedService.GetCommonName()
			expected := certificate.CommonName("service-name-here.namespace-here.svc.cluster.local")
			Expect(actual).To(Equal(expected))
		})
	})
})
