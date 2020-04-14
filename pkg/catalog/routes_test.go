package catalog

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/open-service-mesh/osm/pkg/endpoint"
)

var _ = Describe("UniqueLists", func() {
	Context("Testing uniqueness of services", func() {
		It("Returns unique list of services", func() {

			services := []endpoint.NamespacedService{
				{Namespace: "osm", Service: "booktore-1"},
				{Namespace: "osm", Service: "booktore-1"},
				{Namespace: "osm", Service: "booktore-2"},
				{Namespace: "osm", Service: "booktore-3"},
				{Namespace: "osm", Service: "booktore-2"},
			}

			actual := uniqueServices(services)
			expected := []endpoint.NamespacedService{
				{Namespace: "osm", Service: "booktore-1"},
				{Namespace: "osm", Service: "booktore-2"},
				{Namespace: "osm", Service: "booktore-3"},
			}
			Expect(actual).To(Equal(expected))
		})
	})
})

var _ = Describe("ServicesToString", func() {
	Context("Testing servicesToString", func() {
		It("Returns string list", func() {

			services := []endpoint.NamespacedService{
				{Namespace: "osm", Service: "bookstore-1"},
				{Namespace: "osm", Service: "bookstore-2"},
			}

			actual := servicesToString(services)
			expected := []string{
				"osm/bookstore-1",
				"osm/bookstore-2",
			}
			Expect(actual).To(Equal(expected))
		})
	})
})
