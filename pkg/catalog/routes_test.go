package catalog

import (
	"github.com/deislabs/smc/pkg/endpoint"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var uniqueness = Describe("UniqueLists", func() {
	Context("Testing uniqueness", func() {
		It("Returns unique list of services", func() {

			services := []endpoint.ServiceName{
				endpoint.ServiceName("smc/bookstore-1"),
				endpoint.ServiceName("smc/bookstore-1"),
				endpoint.ServiceName("smc/bookstore-2"),
				endpoint.ServiceName("smc/bookstore-3"),
				endpoint.ServiceName("smc/bookstore-2"),
			}

			actual := uniques(services)
			expected := []endpoint.ServiceName{
				endpoint.ServiceName("smc/bookstore-1"),
				endpoint.ServiceName("smc/bookstore-2"),
				endpoint.ServiceName("smc/bookstore-3"),
			}
			Expect(actual).To(Equal(expected))
		})
	})
})

var serviceToString = Describe("ServicesToString", func() {
	Context("Testing servicesToString", func() {
		It("Returns string list", func() {

			services := []endpoint.ServiceName{
				endpoint.ServiceName("smc/bookstore-1"),
				endpoint.ServiceName("smc/bookstore-2"),
			}

			actual := servicesToString(services)
			expected := []string{
				"smc/bookstore-1",
				"smc/bookstore-2",
			}
			Expect(actual).To(Equal(expected))
		})
	})
})
