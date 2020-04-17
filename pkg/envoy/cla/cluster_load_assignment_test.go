package cla

import (
	"net"

	"github.com/open-service-mesh/osm/pkg/endpoint"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Testing Cluster Load Assignment", func() {
	Context("Testing NewClusterLoadAssignemnt", func() {
		It("Returns cluster load assignment", func() {
			serviceEndpoints := []endpoint.WeightedServiceEndpoints{}
			serviceEndpoints = append(serviceEndpoints, endpoint.WeightedServiceEndpoints{
				WeightedService: endpoint.WeightedService{
					ServiceName: endpoint.NamespacedService{Namespace: "osm", Service: "bookstore-1"},
					Weight:      50,
				},
				Endpoints: []endpoint.Endpoint{
					endpoint.Endpoint{IP: net.IP("0.0.0.0")},
				},
			})
			serviceEndpoints = append(serviceEndpoints, endpoint.WeightedServiceEndpoints{
				WeightedService: endpoint.WeightedService{
					ServiceName: endpoint.NamespacedService{Namespace: "osm", Service: "bookstore-2"},
					Weight:      50,
				},
				Endpoints: []endpoint.Endpoint{
					endpoint.Endpoint{IP: net.IP("0.0.0.1")},
					endpoint.Endpoint{IP: net.IP("0.0.0.2")},
				},
			})

			cla := NewClusterLoadAssignment(serviceEndpoints[0])
			Expect(cla).NotTo(Equal(nil))
			Expect(cla.ClusterName).To(Equal("osm/bookstore-1"))
			Expect(len(cla.Endpoints)).To(Equal(1))
			Expect(len(cla.Endpoints[0].LbEndpoints)).To(Equal(1))
			Expect(cla.Endpoints[0].LbEndpoints[0].GetLoadBalancingWeight().Value).To(Equal(uint32(100)))
			cla2 := NewClusterLoadAssignment(serviceEndpoints[1])
			Expect(cla2).NotTo(Equal(nil))
			Expect(cla2.ClusterName).To(Equal("osm/bookstore-2"))
			Expect(len(cla2.Endpoints)).To(Equal(1))
			Expect(len(cla2.Endpoints[0].LbEndpoints)).To(Equal(2))
			Expect(cla2.Endpoints[0].LbEndpoints[0].GetLoadBalancingWeight().Value).To(Equal(uint32(50)))
			Expect(cla2.Endpoints[0].LbEndpoints[1].GetLoadBalancingWeight().Value).To(Equal(uint32(50)))
		})
	})
})
