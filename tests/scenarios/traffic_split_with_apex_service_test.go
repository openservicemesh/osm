package scenarios

import (
	"fmt"
	"testing"

	xds_route "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	"github.com/golang/mock/gomock"
	"github.com/golang/protobuf/ptypes/wrappers"
	"github.com/stretchr/testify/assert"
	testclient "k8s.io/client-go/kubernetes/fake"

	configv1alpha3 "github.com/openservicemesh/osm/pkg/apis/config/v1alpha3"
	configFake "github.com/openservicemesh/osm/pkg/gen/client/config/clientset/versioned/fake"

	catalogFake "github.com/openservicemesh/osm/pkg/catalog/fake"
	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/envoy/rds"
	"github.com/openservicemesh/osm/pkg/envoy/registry"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/tests"
)

func TestRDSNewResponseWithTrafficSplit(t *testing.T) {
	a := assert.New(t)

	// ---[  Setup the test context  ]---------
	mockCtrl := gomock.NewController(t)
	kubeClient := testclient.NewSimpleClientset()
	configClient := configFake.NewSimpleClientset()
	meshCatalog := catalogFake.NewFakeMeshCatalog(kubeClient, configClient)
	mockConfigurator := configurator.NewMockConfigurator(mockCtrl)

	proxyCertCommonName := certificate.CommonName(fmt.Sprintf("%s.%s.%s.%s", tests.ProxyUUID, envoy.KindSidecar, tests.BookbuyerServiceAccountName, tests.Namespace))
	proxyCertSerialNumber := certificate.SerialNumber("123456")
	proxy, err := getProxy(kubeClient, proxyCertCommonName, proxyCertSerialNumber)
	a.Nil(err)
	a.NotNil(proxy)

	proxyRegistry := registry.NewProxyRegistry(registry.ExplicitProxyServiceMapper(func(*envoy.Proxy) ([]service.MeshService, error) {
		return nil, nil
	}), nil)

	// ---[  Get the config from rds.NewResponse()  ]-------
	mockConfigurator.EXPECT().IsPermissiveTrafficPolicyMode().Return(false).AnyTimes()

	mockConfigurator.EXPECT().GetFeatureFlags().Return(configv1alpha3.FeatureFlags{
		EnableWASMStats:    false,
		EnableEgressPolicy: false,
	}).AnyTimes()

	resources, err := rds.NewResponse(meshCatalog, proxy, nil, mockConfigurator, nil, proxyRegistry)
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
