package kube

import (
	"net"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/openservicemesh/osm/pkg/endpoint"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/tests"
)

var _ = Describe("Test Kubernetes Provider", func() {
	Context("Testing ListServicesForServiceAccount", func() {
		It("returns empty list", func() {
			c := NewFakeProvider()
			svc := service.NamespacedService{
				Service: "blah",
			}
			Ω(func() { c.ListEndpointsForService(svc) }).Should(Panic())

		})

		It("returns the services for a given service account", func() {
			c := NewFakeProvider()
			actual := c.ListEndpointsForService(tests.BookstoreService)
			expected := []endpoint.Endpoint{{
				IP:   net.ParseIP("8.8.8.8"),
				Port: 8888,
			}}
			Expect(actual).To(Equal(expected))
		})
	})
})
