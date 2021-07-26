package scenarios

import (
	"fmt"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/tests"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	xds_route "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	"github.com/golang/mock/gomock"
	"github.com/onsi/ginkgo"
	testclient "k8s.io/client-go/kubernetes/fake"

	"github.com/openservicemesh/osm/pkg/apis/config/v1alpha1"
	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/envoy/rds"
	"github.com/openservicemesh/osm/pkg/envoy/registry"
	configFake "github.com/openservicemesh/osm/pkg/gen/client/config/clientset/versioned/fake"
	"github.com/openservicemesh/osm/pkg/service"
)

var _ = Describe(``+

	`This is a test of the RDS configuration created for the bookbuyer Envoy in a setup `+
	`where bookstore traffic is split between a bookstore-v1 and bookstore-v2 services. `+
	`But we also have the v1 pods match the labels on the bookstore-apex service as well.`,

	func() {
		Context("Test rds.NewResponse()", func() {

			// ---[  Setup the test context  ]---------
			mockCtrl := gomock.NewController(ginkgo.GinkgoT())
			kubeClient := testclient.NewSimpleClientset()
			configClient := configFake.NewSimpleClientset()
			meshCatalog := catalog.NewFakeMeshCatalog(kubeClient, configClient)
			mockConfigurator := configurator.NewMockConfigurator(mockCtrl)

			proxyCertCommonName := certificate.CommonName(fmt.Sprintf("%s.%s.%s.%s", tests.ProxyUUID, envoy.KindSidecar, tests.BookbuyerServiceAccountName, tests.Namespace))
			proxyCertSerialNumber := certificate.SerialNumber("123456")
			proxy, err := getProxy(kubeClient, proxyCertCommonName, proxyCertSerialNumber)
			It("sets up test context - SMI policies, Services, Pods etc.", func() {
				Expect(err).ToNot(HaveOccurred())
				Expect(meshCatalog).ToNot(BeNil())
				Expect(proxy).ToNot(BeNil())
			})

			proxyRegistry := registry.NewProxyRegistry(registry.ExplicitProxyServiceMapper(func(*envoy.Proxy) ([]service.MeshService, error) {
				return nil, nil
			}))

			// ---[  Get the config from rds.NewResponse()  ]-------
			mockConfigurator.EXPECT().IsPermissiveTrafficPolicyMode().Return(false).AnyTimes()

			mockConfigurator.EXPECT().GetFeatureFlags().Return(v1alpha1.FeatureFlags{
				EnableWASMStats:    false,
				EnableEgressPolicy: false,
			}).AnyTimes()

			resources, err := rds.NewResponse(meshCatalog, proxy, nil, mockConfigurator, nil, proxyRegistry)
			It("did not return an error", func() {
				Expect(err).ToNot(HaveOccurred())
				Expect(resources).ToNot(BeNil())
				Expect(len(resources)).To(Equal(2))
			})

			// ---[  Prepare the config for testing  ]-------
			// Order matters, inbound is returned always in first index, outbound second one
			routeCfg, ok := resources[1].(*xds_route.RouteConfiguration)
			It("returns a response that can be unmarshalled into an xds RouteConfiguration struct", func() {
				Expect(ok).To(BeTrue())
				Expect(routeCfg.Name).To(Equal("rds-outbound"))
			})

			const (
				apexName = "outbound_virtual-host|bookstore-apex.default.local"
				v1Name   = "outbound_virtual-host|bookstore-v1.default.local"
				v2Name   = "outbound_virtual-host|bookstore-v2.default.local"
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
			for _, virtualHost := range routeCfg.VirtualHosts {
				map[string]func(){
					apexName: func() { apex = virtualHost },
					v1Name:   func() { v1 = virtualHost },
					v2Name:   func() { v2 = virtualHost },
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
						weightedCluster("bookstore-v1", 100),
					},
					TotalWeight: toInt(100),
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
						weightedCluster("bookstore-v2", 100),
					},
					TotalWeight: toInt(100),
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
						// weightedCluster("bookstore-apex", 100),
						weightedCluster("bookstore-v1", 90),
						weightedCluster("bookstore-v2", 10),
					},
					TotalWeight: toInt(100),
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
