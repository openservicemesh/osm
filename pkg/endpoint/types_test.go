package endpoint

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/open-service-mesh/osm/pkg/certificate"
)

var _ = Describe("UniqueLists", func() {
	Context("Testing uniqueness of services", func() {
		It("Returns unique list of services", func() {
			namespacedService := NamespacedService{
				Namespace: "namespace-here",
				Service:   "service-name-here",
			}
			actual := namespacedService.GetCommonName()
			expected := certificate.CommonName("service-name-here.namespace-here.svc.cluster.local")
			Expect(actual).To(Equal(expected))
		})
	})
})
