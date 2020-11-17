package catalog

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/openservicemesh/osm/pkg/endpoint"
	"github.com/openservicemesh/osm/pkg/tests"
)

var _ = Describe("Test catalog functions", func() {
	mc := newFakeMeshCatalog()
	Context("Testing ListEndpointsForService()", func() {
		It("lists endpoints for a given service", func() {
			actual, err := mc.ListEndpointsForService(tests.BookstoreV1Service)
			Expect(err).ToNot(HaveOccurred())

			expected := []endpoint.Endpoint{
				tests.Endpoint,
			}
			Expect(actual).To(Equal(expected))
		})
	})

	Context("Testing GetResolvableServiceEndpoints()", func() {
		It("returns the endpoint for the service", func() {
			actual, err := mc.GetResolvableServiceEndpoints(tests.BookstoreV1Service)
			Expect(err).ToNot(HaveOccurred())

			expected := []endpoint.Endpoint{
				tests.Endpoint,
			}
			Expect(actual).To(Equal(expected))
		})
	})

})
