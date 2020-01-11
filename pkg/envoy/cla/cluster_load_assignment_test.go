package cla

import (
	"github.com/deislabs/smc/pkg/mesh"
	
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Cluster Load Assignment", func() {
	Describe("Testing Cluster Load Assignment", func() {
		Context("Testing NewClusterLoadAssignemnt", func() {
			It("Returns cluster load assignment", func() {
				weightedServices := []mesh.WeightedService{}
				weightedServices = append(weightedServices, mesh.WeightedService{
					ServiceName: mesh.ServiceName("bookstore-1"),
					Weight:      50,
					IPs:         []mesh.IP{"0.0.0.0", "0.0.0.1"},
				})
				weightedServices = append(weightedServices, mesh.WeightedService{
					ServiceName: mesh.ServiceName("bookstore-2"),
					Weight:      50,
					IPs:         []mesh.IP{"0.0.0.2"},
				})
				cla := NewClusterLoadAssignment("bookstore", weightedServices)
				Expect(cla).NotTo(Equal(nil))
				Expect(cla.ClusterName).To(Equal("bookstore"))
				Expect(len(cla.Endpoints)).To(Equal(1))
				Expect(len(cla.Endpoints[0].LbEndpoints)).To(Equal(3))
				Expect(cla.Endpoints[0].LbEndpoints[0].GetLoadBalancingWeight().Value).To(Equal(uint32(25)))
				Expect(cla.Endpoints[0].LbEndpoints[1].GetLoadBalancingWeight().Value).To(Equal(uint32(25)))
				Expect(cla.Endpoints[0].LbEndpoints[2].GetLoadBalancingWeight().Value).To(Equal(uint32(50)))
				

			})
		})
	})
})
