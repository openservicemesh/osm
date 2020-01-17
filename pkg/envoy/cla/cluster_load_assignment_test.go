package cla

import (
	"net"

	"github.com/deislabs/smc/pkg/endpoint"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Cluster Load Assignment", func() {
	Describe("Testing Cluster Load Assignment", func() {
		Context("Testing NewClusterLoadAssignemnt", func() {
			It("Returns cluster load assignment", func() {
				weightedServices := []endpoint.WeightedService{}
				weightedServices = append(weightedServices, endpoint.WeightedService{
					ServiceName: endpoint.ServiceName("bookstore-1"),
					Weight:      50,
					Endpoints:   []endpoint.Endpoint{endpoint.Endpoint{IP: net.IP("0.0.0.0")}},
				})
				weightedServices = append(weightedServices, endpoint.WeightedService{
					ServiceName: endpoint.ServiceName("bookstore-2"),
					Weight:      50,
					Endpoints:   []endpoint.Endpoint{endpoint.Endpoint{IP: net.IP("0.0.0.1")}, endpoint.Endpoint{IP: net.IP("0.0.0.2")}},
				})

				cla := NewClusterLoadAssignment("bookstore", weightedServices)
				Expect(cla).NotTo(Equal(nil))
				Expect(cla.ClusterName).To(Equal("bookstore"))
				Expect(len(cla.Endpoints)).To(Equal(1))
				Expect(len(cla.Endpoints[0].LbEndpoints)).To(Equal(3))
				Expect(cla.Endpoints[0].LbEndpoints[0].GetLoadBalancingWeight().Value).To(Equal(uint32(50)))
				Expect(cla.Endpoints[0].LbEndpoints[1].GetLoadBalancingWeight().Value).To(Equal(uint32(25)))
				Expect(cla.Endpoints[0].LbEndpoints[2].GetLoadBalancingWeight().Value).To(Equal(uint32(25)))
			})
		})
	})
})
