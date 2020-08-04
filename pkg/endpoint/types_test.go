package endpoint

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/service"
)

var _ = Describe("Test MeshService methods", func() {
	Context("Testing GetCommonName", func() {
		It("should return DNS-1123 of the MeshService struct", func() {
			meshService := service.MeshService{
				Namespace: "namespace-here",
				Name:      "service-name-here",
			}
			actual := meshService.GetCommonName()
			expected := certificate.CommonName("service-name-here.namespace-here.svc.cluster.local")
			Expect(actual).To(Equal(expected))
		})
	})
})
