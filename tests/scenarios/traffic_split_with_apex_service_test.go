package scenarios

import (
	"testing"
	"time"

	xds_route "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	"github.com/golang/mock/gomock"
	"github.com/golang/protobuf/ptypes/wrappers"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	testclient "k8s.io/client-go/kubernetes/fake"

	catalogFake "github.com/openservicemesh/osm/pkg/catalog/fake"
	tresorFake "github.com/openservicemesh/osm/pkg/certificate/providers/tresor/fake"
	"github.com/openservicemesh/osm/pkg/compute"
	"github.com/openservicemesh/osm/pkg/compute/kube"
	"github.com/openservicemesh/osm/pkg/endpoint"
	"github.com/openservicemesh/osm/pkg/envoy/rds"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/tests"
)

func TestRDSNewResponseWithTrafficSplit(t *testing.T) {
	a := assert.New(t)

	// ---[  Setup the test context  ]---------
	kubeClient := testclient.NewSimpleClientset()
	mockCtrl := gomock.NewController(t)
	services := []service.MeshService{tests.BookstoreApexService, tests.BookstoreV1Service, tests.BookstoreV2Service}
	provider := compute.NewMockInterface(mockCtrl)
	provider.EXPECT().GetMeshConfig().AnyTimes()
	provider.EXPECT().GetServicesForServiceIdentity(gomock.Any()).Return(services).AnyTimes()
	provider.EXPECT().GetResolvableEndpointsForService(gomock.Any()).Return([]endpoint.Endpoint{tests.Endpoint}).AnyTimes()
	provider.EXPECT().GetTargetPortForServicePort(gomock.Any(), gomock.Any()).Return(uint16(8888), nil).AnyTimes()
	provider.EXPECT().ListServicesForProxy(gomock.Any()).Return(nil, nil).AnyTimes()

	provider.EXPECT().ListEgressPoliciesForServiceAccount(gomock.Any()).Return(nil).AnyTimes()
	provider.EXPECT().GetIngressBackendPolicyForService(gomock.Any()).Return(nil).AnyTimes()
	provider.EXPECT().GetUpstreamTrafficSettingByService(gomock.Any()).Return(nil).AnyTimes()
	provider.EXPECT().GetUpstreamTrafficSettingByNamespace(gomock.Any()).Return(nil).AnyTimes()
	for _, svc := range services {
		provider.EXPECT().GetHostnamesForService(svc, true).Return(kube.NewClient(nil).GetHostnamesForService(svc, true)).AnyTimes()
	}

	meshCatalog := catalogFake.NewFakeMeshCatalog(provider)

	proxy, err := getSidecarProxy(kubeClient, uuid.MustParse(tests.ProxyUUID), identity.New(tests.BookbuyerServiceAccountName, tests.Namespace))
	a.Nil(err)
	a.NotNil(proxy)

	mc := tresorFake.NewFake(1 * time.Hour)
	a.NotNil(a)

	resources, err := rds.NewResponse(meshCatalog, proxy, mc, nil)
	a.Nil(err)
	a.Len(resources, 1) // only outbound routes configured for this test

	// ---[  Prepare the config for testing  ]-------
	// Order matters. In this test, we do not expect rds-inbound route configuration, and rds-outbound is expected
	// to be configured per outbound port.
	routeCfg, ok := resources[0].(*xds_route.RouteConfiguration)
	a.True(ok)
	a.Equal("rds-outbound.8888", routeCfg.Name)

	const (
		apexName = "outbound_virtual-host|bookstore-apex.default.svc.cluster.local"
		v1Name   = "outbound_virtual-host|bookstore-v1.default.svc.cluster.local"
		v2Name   = "outbound_virtual-host|bookstore-v2.default.svc.cluster.local"
	)
	expectedVHostNames := []string{apexName, v1Name, v2Name}

	// ---[  Compare with expectations  ]-------
	// Expect an XDS Route Configuration with 3 outbound virtual hosts"
	var actualNames []string
	for _, vHost := range routeCfg.VirtualHosts {
		actualNames = append(actualNames, vHost.Name)
	}
	a.Len(routeCfg.VirtualHosts, len(expectedVHostNames))
	a.ElementsMatch(expectedVHostNames, actualNames)

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

	testCases := []struct {
		name                    string
		virtualHost             *xds_route.VirtualHost
		expectedDomains         []string
		expectedWeightedCluster *xds_route.WeightedCluster
	}{
		{
			name:        "bookstore-v1",
			virtualHost: v1,
			expectedDomains: []string{
				"bookstore-v1",
				"bookstore-v1:8888",
				"bookstore-v1.default",
				"bookstore-v1.default.svc",
				"bookstore-v1.default:8888",
				"bookstore-v1.default.svc:8888",
				"bookstore-v1.default.svc.cluster",
				"bookstore-v1.default.svc.cluster:8888",
				"bookstore-v1.default.svc.cluster.local",
				"bookstore-v1.default.svc.cluster.local:8888",
			},
			expectedWeightedCluster: &xds_route.WeightedCluster{
				Clusters: []*xds_route.WeightedCluster_ClusterWeight{
					{
						Name: "default/bookstore-v1|8888",
						Weight: &wrappers.UInt32Value{
							Value: 100,
						},
					},
				},
				TotalWeight: &wrappers.UInt32Value{
					Value: 100,
				},
			},
		},
		{
			name:        "bookstore-v2",
			virtualHost: v2,
			expectedDomains: []string{
				"bookstore-v2",
				"bookstore-v2:8888",
				"bookstore-v2.default",
				"bookstore-v2.default.svc",
				"bookstore-v2.default:8888",
				"bookstore-v2.default.svc:8888",
				"bookstore-v2.default.svc.cluster",
				"bookstore-v2.default.svc.cluster:8888",
				"bookstore-v2.default.svc.cluster.local",
				"bookstore-v2.default.svc.cluster.local:8888",
			},
			expectedWeightedCluster: &xds_route.WeightedCluster{
				Clusters: []*xds_route.WeightedCluster_ClusterWeight{
					{
						Name: "default/bookstore-v2|8888",
						Weight: &wrappers.UInt32Value{
							Value: 100,
						},
					},
				},
				TotalWeight: &wrappers.UInt32Value{
					Value: 100,
				},
			},
		},
		{
			name:        "bookstore-apex",
			virtualHost: apex,
			expectedDomains: []string{
				"bookstore-apex",
				"bookstore-apex:8888",
				"bookstore-apex.default",
				"bookstore-apex.default.svc",
				"bookstore-apex.default:8888",
				"bookstore-apex.default.svc:8888",
				"bookstore-apex.default.svc.cluster",
				"bookstore-apex.default.svc.cluster:8888",
				"bookstore-apex.default.svc.cluster.local",
				"bookstore-apex.default.svc.cluster.local:8888",
			},
			expectedWeightedCluster: &xds_route.WeightedCluster{
				Clusters: []*xds_route.WeightedCluster_ClusterWeight{
					{
						Name: "default/bookstore-v1|8888",
						Weight: &wrappers.UInt32Value{
							Value: 90,
						},
					},
					{
						Name: "default/bookstore-v2|8888",
						Weight: &wrappers.UInt32Value{
							Value: 10,
						},
					},
				},
				TotalWeight: &wrappers.UInt32Value{
					Value: 100,
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			a.ElementsMatch(tc.expectedDomains, tc.virtualHost.Domains)
			a.Len(tc.virtualHost.Routes, 1)
			a.ElementsMatch(tc.expectedWeightedCluster.Clusters, tc.virtualHost.Routes[0].GetRoute().GetWeightedClusters().Clusters)
			a.Equal(tc.expectedWeightedCluster.TotalWeight, tc.virtualHost.Routes[0].GetRoute().GetWeightedClusters().TotalWeight)
		})
	}
}
