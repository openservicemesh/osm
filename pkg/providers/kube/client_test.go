package kube

import (
	"net"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/open-service-mesh/osm/pkg/endpoint"
)

var _ = Describe("Test Kubernetes Provider", func() {

	Context("Testing ListServicesForServiceAccount", func() {
		It("returns empty list", func() {
			c := NewFakeProvider()
			Ω(func() { c.ListEndpointsForService("blah") }).Should(Panic())

		})

		It("returns the services for a given service account", func() {
			c := NewFakeProvider()
			actual := c.ListEndpointsForService("default/bookstore")
			expected := []endpoint.Endpoint{{
				IP:   net.ParseIP("8.8.8.8"),
				Port: 8888,
			}}
			Expect(actual).To(Equal(expected))
		})
	})
})
