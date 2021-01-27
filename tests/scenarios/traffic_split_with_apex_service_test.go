package scenarios

import (
	"fmt"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/kubernetes"
	testclient "k8s.io/client-go/kubernetes/fake"

	xds_route "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/wrappers"

	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/envoy/rds"
	"github.com/openservicemesh/osm/pkg/tests"
)

var _ = Describe(``+

	`This is a test of the RDS configuration created for the bookbuyer Envoy in a setup `+
	`where bookstore traffic is split between a bookstore-v1 and bookstore-v2 services. `+
	`But we also have the v1 pods match the labels on the bookstore-apex service as well.`,

	func() {
		Context("Test rds.NewResponse()", func() {

			// ---[  Setup the test context  ]---------
			kubeClient := testclient.NewSimpleClientset()
			meshCatalog := catalog.NewFakeMeshCatalog(kubeClient)
			proxy, err := getProxy(kubeClient)
			It("sets up test context - SMI policies, Services, Pods etc.", func() {
				Expect(err).ToNot(HaveOccurred())
				Expect(meshCatalog).ToNot(BeNil())
				Expect(proxy).ToNot(BeNil())
			})

			// ---[  Get the config from rds.NewResponse()  ]-------

			actual, err := rds.NewResponse(meshCatalog, proxy, nil, nil, nil)
			It("did not return an error", func() {
				Expect(err).ToNot(HaveOccurred())
				Expect(actual).ToNot(BeNil())
				Expect(len(actual.Resources)).To(Equal(2))
			})

			// ---[  Prepare the config for testing  ]-------
			routeCfg := xds_route.RouteConfiguration{}

			err = ptypes.UnmarshalAny(actual.Resources[0], &routeCfg)
			It("returns a response that can be unmarshalled into an xds RouteConfiguration struct", func() {
				Expect(err).ToNot(HaveOccurred())
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

func getProxy(kubeClient kubernetes.Interface) (*envoy.Proxy, error) {
	bookbuyerPodLabels := map[string]string{
		tests.SelectorKey:                tests.BookbuyerService.Name,
		constants.EnvoyUniqueIDLabelName: tests.ProxyUUID,
	}
	if _, err := tests.MakePod(kubeClient, tests.Namespace, tests.BookbuyerServiceName, tests.BookbuyerServiceAccountName, bookbuyerPodLabels); err != nil {
		return nil, err
	}

	bookstorePodLabels := map[string]string{
		tests.SelectorKey:                "bookstore",
		constants.EnvoyUniqueIDLabelName: uuid.New().String(),
	}
	if _, err := tests.MakePod(kubeClient, tests.Namespace, "bookstore", tests.BookstoreServiceAccountName, bookstorePodLabels); err != nil {
		return nil, err
	}

	selectors := map[string]string{
		tests.SelectorKey: tests.BookbuyerServiceName,
	}
	if _, err := tests.MakeService(kubeClient, tests.BookbuyerServiceName, selectors); err != nil {
		return nil, err
	}

	for _, svcName := range []string{tests.BookstoreApexServiceName, tests.BookstoreV1ServiceName, tests.BookstoreV2ServiceName} {
		selectors := map[string]string{
			tests.SelectorKey: "bookstore",
		}
		if _, err := tests.MakeService(kubeClient, svcName, selectors); err != nil {
			return nil, err
		}
	}

	certCommonName := certificate.CommonName(fmt.Sprintf("%s.%s.%s", tests.ProxyUUID, tests.BookbuyerServiceAccountName, tests.Namespace))
	certSerialNumber := certificate.SerialNumber("123456")
	proxy := envoy.NewProxy(certCommonName, certSerialNumber, nil)
	return proxy, nil
}
