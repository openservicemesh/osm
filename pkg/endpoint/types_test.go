package endpoint

import (
	"net"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Test Endpoint methods", func() {
	Context("Testing Endpoint{}.String()", func() {
		It("should return a proper string representation of the Endpoint struct", func() {
			actual := Endpoint{
				IP:   net.ParseIP("8.8.8.8"),
				Port: Port(8888),
			}.String()
			expected := "(ip=8.8.8.8, port=8888)"
			Expect(actual).To(Equal(expected))
		})
	})
})
