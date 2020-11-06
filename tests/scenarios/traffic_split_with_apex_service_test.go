package scenarios

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	xds_route "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	xds_discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/wrappers"

	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/envoy/rds"
)

var _ = Describe(``+

	`This is a test of the RDS configuration created for the bookbuyer Envoy in a setup `+
	`where bookstore traffic is split between a bookstore-v1 and bookstore-v2 services. `+
	`But we also have the v1 pods match the labels on the bookstore-apex service as well.`,

	func() {
		Context("Test rds.NewResponse()", func() {

			// ---[  Setup the test context  ]---------
			var meshCatalog catalog.MeshCataloger
			var proxy *envoy.Proxy

			It("sets up test context - SMI policies, Services, Pods etc.", func() {
				var err error
				meshCatalog, proxy, err = getMeshCatalogAndProxy()
				Expect(err).ToNot(HaveOccurred())
			})

			// ---[  Get the config from rds.NewResponse()  ]-------
			var actual *xds_discovery.DiscoveryResponse
			It("did not return an error", func() {
				var err error
				actual, err = rds.NewResponse(meshCatalog, proxy, nil, nil, nil)
				Expect(err).ToNot(HaveOccurred())
			})

			// ---[  Prepare the config for testing  ]-------
			routeCfg := xds_route.RouteConfiguration{}

			It("returns a response that can be unmarshalled into an xds RouteConfiguration struct", func() {
				err := ptypes.UnmarshalAny(actual.Resources[0], &routeCfg)
				Expect(err).ToNot(HaveOccurred())
			})

			It("created an XDS Route Configuration with the correct name", func() {
				Expect(routeCfg.Name).To(Equal("RDS_Outbound"))
			})

			const (
				apexName = "outbound_virtualHost|bookstore-apex"
				v1Name   = "outbound_virtualHost|bookstore-v1"
				v2Name   = "outbound_virtualHost|bookstore-v2"
			)
			expectedVHostNames := []string{apexName, v1Name, v2Name}

			// ---[  Compare with expectations  ]-------
			It("created an XDS Route Configuration with 3 outbound virtual hosts", func() {
				var actualNames []string
				for _, vHost := range routeCfg.VirtualHosts {
					actualNames = append(actualNames, vHost.Name)
				}
				Expect(len(routeCfg.VirtualHosts)).To(Equal(len(expectedVHostNames)), fmt.Sprintf("Here are the actual virtual hosts: %+v", actualNames))
				Expect(actualNames).To(ContainElements(expectedVHostNames))

			})

			// Get the 3 VirtualHost configurations into variables so it is easier to
			// test them (they are stored in a slice w/ non-deterministic order)
			var apex, v1, v2 *xds_route.VirtualHost
			for idx, virtualHost := range routeCfg.VirtualHosts {
				map[string]func(){
					apexName: func() { apex = routeCfg.VirtualHosts[idx] },
					v1Name:   func() { v1 = routeCfg.VirtualHosts[idx] },
					v2Name:   func() { v2 = routeCfg.VirtualHosts[idx] },
				}[virtualHost.Name]()
			}

			It("created correct 'bookstore-v1' XDS Route Configuration", func() {
				expectedDomains := []string{
					"bookstore-v1",
					"bookstore-v1.default",
					"bookstore-v1.default.svc",
					"bookstore-v1.default.svc.cluster",
					"bookstore-v1.default.svc.cluster.local",

					"bookstore-v1:8888",
					"bookstore-v1.default:8888",
					"bookstore-v1.default.svc:8888",
					"bookstore-v1.default.svc.cluster:8888",
					"bookstore-v1.default.svc.cluster.local:8888",
				}

				expectedWeightedCluster := &xds_route.WeightedCluster{
					Clusters: []*xds_route.WeightedCluster_ClusterWeight{
						weightedCluster("bookstore-v1", 90),
					},
					TotalWeight: toInt(90),
				}

				checkExpectations(expectedDomains, expectedWeightedCluster, v1)
			})

			It("created correct 'bookstore-v2' XDS Route Configuration", func() {
				expectedDomains := []string{
					"bookstore-v2",
					"bookstore-v2.default",
					"bookstore-v2.default.svc",
					"bookstore-v2.default.svc.cluster",
					"bookstore-v2.default.svc.cluster.local",

					"bookstore-v2:8888",
					"bookstore-v2.default:8888",
					"bookstore-v2.default.svc:8888",
					"bookstore-v2.default.svc.cluster:8888",
					"bookstore-v2.default.svc.cluster.local:8888",
				}

				expectedWeightedCluster := &xds_route.WeightedCluster{
					Clusters: []*xds_route.WeightedCluster_ClusterWeight{
						weightedCluster("bookstore-v2", 10),
					},
					TotalWeight: toInt(10),
				}

				checkExpectations(expectedDomains, expectedWeightedCluster, v2)
			})

			It("created correct 'bookstore-apex' XDS Route Configuration", func() {
				expectedDomains := []string{
					"bookstore-apex",
					"bookstore-apex.default",
					"bookstore-apex.default.svc",
					"bookstore-apex.default.svc.cluster",
					"bookstore-apex.default.svc.cluster.local",

					"bookstore-apex:8888",
					"bookstore-apex.default:8888",
					"bookstore-apex.default.svc:8888",
					"bookstore-apex.default.svc.cluster:8888",
					"bookstore-apex.default.svc.cluster.local:8888",
				}

				expectedWeightedCluster := &xds_route.WeightedCluster{
					Clusters: []*xds_route.WeightedCluster_ClusterWeight{
						weightedCluster("bookstore-apex", 100),
						weightedCluster("bookstore-v1", 90),
						weightedCluster("bookstore-v2", 10),
					},
					TotalWeight: toInt(200),
				}

				checkExpectations(expectedDomains, expectedWeightedCluster, apex)
			})
		})
	})

func checkExpectations(expectedDomains []string, expectedWeightedCluster *xds_route.WeightedCluster, virtualHost *xds_route.VirtualHost) {
	Expect(len(virtualHost.Domains)).To(Equal(len(expectedDomains)))
	Expect(virtualHost.Domains).To(ContainElements(expectedDomains))

	Expect(len(virtualHost.Routes)).To(Equal(1))

	Expect(len(virtualHost.Routes[0].GetRoute().GetWeightedClusters().Clusters)).To(Equal(len(expectedWeightedCluster.Clusters)))
	Expect(virtualHost.Routes[0].GetRoute().GetWeightedClusters().Clusters[0]).To(Equal(expectedWeightedCluster.Clusters[0]))
	Expect(virtualHost.Routes[0].GetRoute().GetWeightedClusters().TotalWeight).To(Equal(expectedWeightedCluster.TotalWeight))
}

func toInt(val uint32) *wrappers.UInt32Value {
	return &wrappers.UInt32Value{
		Value: val,
	}
}

func weightedCluster(serviceName string, weight uint32) *xds_route.WeightedCluster_ClusterWeight {
	return &xds_route.WeightedCluster_ClusterWeight{
		Name:   fmt.Sprintf("default/%s", serviceName),
		Weight: toInt(weight),
	}
}
