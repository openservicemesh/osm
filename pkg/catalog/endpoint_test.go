package catalog

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/openservicemesh/osm/pkg/endpoint"
	"github.com/openservicemesh/osm/pkg/tests"
)

var _ = Describe("Test catalog functions", func() {
	Context("Testing ListEndpointsForService()", func() {
		mc := newFakeMeshCatalog()
		It("lists endpoints for a given service", func() {
			actual, err := mc.ListEndpointsForService(tests.BookstoreService)
			Expect(err).ToNot(HaveOccurred())

			expected := []endpoint.Endpoint{
				tests.Endpoint,
			}
			Expect(actual).To(Equal(expected))
		})
	})
})
