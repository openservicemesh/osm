package catalog

import (
	"net"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/open-service-mesh/osm/pkg/endpoint"
)

var _ = Describe("Endpoints To String", func() {
	Context("Testing endpointsToString", func() {
		It("Returns string representation of a list of endpoints", func() {

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
			expected := "(ip=10.20.30.1, port=10),(ip=210.220.230.21, port=202)"
			Expect(actual).To(Equal(expected))
		})
	})
})
