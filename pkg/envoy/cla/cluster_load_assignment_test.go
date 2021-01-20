package cla

import (
	"net"

	"github.com/openservicemesh/osm/pkg/endpoint"
	"github.com/openservicemesh/osm/pkg/service"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Testing Cluster Load Assignment", func() {
	Context("Testing NewClusterLoadAssignment", func() {
		It("Returns cluster load assignment", func() {

			namespacedServices := []service.MeshService{
				{Namespace: "osm", Name: "bookstore-1"},
				{Namespace: "osm", Name: "bookstore-2"},
			}

			allServiceEndpoints := map[service.MeshService][]endpoint.Endpoint{
				namespacedServices[0]: {
					{IP: net.IP("0.0.0.0")},
				},
				namespacedServices[1]: {
					{IP: net.IP("0.0.0.1")},
					{IP: net.IP("0.0.0.2")},
				},
			}

			cla := NewClusterLoadAssignment(namespacedServices[0], allServiceEndpoints[namespacedServices[0]])
			Expect(cla).NotTo(Equal(nil))
			Expect(cla.ClusterName).To(Equal("osm/bookstore-1"))
			Expect(len(cla.Endpoints)).To(Equal(1))
			Expect(len(cla.Endpoints[0].LbEndpoints)).To(Equal(1))
			Expect(cla.Endpoints[0].LbEndpoints[0].GetLoadBalancingWeight().Value).To(Equal(uint32(100)))
			cla2 := NewClusterLoadAssignment(namespacedServices[1], allServiceEndpoints[namespacedServices[1]])
			Expect(cla2).NotTo(Equal(nil))
			Expect(cla2.ClusterName).To(Equal("osm/bookstore-2"))
			Expect(len(cla2.Endpoints)).To(Equal(1))
			Expect(len(cla2.Endpoints[0].LbEndpoints)).To(Equal(2))
			Expect(cla2.Endpoints[0].LbEndpoints[0].GetLoadBalancingWeight().Value).To(Equal(uint32(50)))
			Expect(cla2.Endpoints[0].LbEndpoints[1].GetLoadBalancingWeight().Value).To(Equal(uint32(50)))
		})
	})
})
