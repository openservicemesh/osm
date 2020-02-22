package catalog

import (
	"net"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/deislabs/smc/pkg/endpoint"
)

var _ = Describe("Endpoints To String", func() {
	Context("Testing endpointsToString", func() {
		It("Returns string list", func() {

			endpoints := []endpoint.Endpoint{
				{
					IP:   net.ParseIP("10.20.30.1"),
					Port: 10,
				},
				{
					IP:   net.ParseIP("210.220.230.21"),
					Port: 202,
				},
			}

			actual := endpointsToString(endpoints)
			expected := []string{
				"10.20.30.1:10",
				"210.220.230.21:202",
			}
			Expect(actual).To(Equal(expected))
		})
	})
})
