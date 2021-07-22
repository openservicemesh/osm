package multicluster

import (
	"fmt"
	"testing"

	xds_route "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	"github.com/golang/mock/gomock"
	tassert "github.com/stretchr/testify/assert"
	testclient "k8s.io/client-go/kubernetes/fake"

	"github.com/openservicemesh/osm/pkg/apis/config/v1alpha1"
	configFake "github.com/openservicemesh/osm/pkg/gen/client/config/clientset/versioned/fake"

	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/envoy/rds"
	"github.com/openservicemesh/osm/pkg/envoy/registry"
	"github.com/openservicemesh/osm/pkg/service"
)

func TestMulticlusterGatewaySecretDiscoveryService(t *testing.T) {
	assert := tassert.New(t)

	checkIt := func(expectedDomains []string, expectedWeightedCluster *xds_route.WeightedCluster, virtualHost *xds_route.VirtualHost) {
		assert.Equal(len(virtualHost.Domains), len(expectedDomains))
		assert.ElementsMatch(virtualHost.Domains, expectedDomains)

		assert.Len(virtualHost.Routes, 1)

		assert.Equal(len(virtualHost.Routes[0].GetRoute().GetWeightedClusters().Clusters), len(expectedWeightedCluster.Clusters))
		assert.Equal(virtualHost.Routes[0].GetRoute().GetWeightedClusters().Clusters[0], expectedWeightedCluster.Clusters[0])
		assert.Equal(virtualHost.Routes[0].GetRoute().GetWeightedClusters().TotalWeight, expectedWeightedCluster.TotalWeight)
	}

	// ---[  Setup the test context  ]---------
	mockCtrl := gomock.NewController(t)
	kubeClient := testclient.NewSimpleClientset()
	configClient := configFake.NewSimpleClientset()
	meshCatalog := catalog.NewFakeMeshCatalog(kubeClient, configClient)
	mockConfigurator := configurator.NewMockConfigurator(mockCtrl)
	proxy, err := getProxy(kubeClient)
	assert.Nil(err)

	proxyRegistry := registry.NewProxyRegistry(registry.ExplicitProxyServiceMapper(func(*envoy.Proxy) ([]service.MeshService, error) {
		return nil, nil
	}))

	// ---[  Get the config from rds.NewResponse()  ]-------
	mockConfigurator.EXPECT().IsPermissiveTrafficPolicyMode().Return(false).AnyTimes()

	mockConfigurator.EXPECT().GetFeatureFlags().Return(v1alpha1.FeatureFlags{
		EnableWASMStats:    false,
		EnableEgressPolicy: false,
		EnableOSMGateway:   true, // ENABLE THE MULTICLUSTER GATEWAY
	}).AnyTimes()

	resources, err := rds.NewResponse(meshCatalog, proxy, nil, mockConfigurator, nil, proxyRegistry)
	assert.Nil(err, fmt.Sprintf("rds.NewResponse return unexpected error: %+v", err))
	assert.NotNil(resources)
	assert.Len(resources, 2)

	// ---[  Prepare the config for testing  ]-------
	// Order matters, inbound is returned always in first index, outbound second one
	routeCfg, ok := resources[1].(*xds_route.RouteConfiguration)

	assert.True(ok)
	assert.Equal(routeCfg.Name, "rds-outbound")

	const (
		apexName = "outbound_virtual-host|bookstore-apex.default.local"
		v1Name   = "outbound_virtual-host|bookstore-v1.default.local"
		v2Name   = "outbound_virtual-host|bookstore-v2.default.local"
	)

	{
		expectedVHostNames := []string{apexName, v1Name, v2Name}

		// ---[  Compare with expectations  ]-------
		// created an XDS Route Configuration with 3 outbound virtual hosts
		var actualNames []string
		for _, vHost := range routeCfg.VirtualHosts {
			actualNames = append(actualNames, vHost.Name)
		}
		assert.Equal(len(routeCfg.VirtualHosts), len(expectedVHostNames), fmt.Sprintf("Here are the actual virtual hosts: %+v", actualNames))
		assert.ElementsMatch(actualNames, expectedVHostNames)
	}

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

	// created correct 'bookstore-v1' XDS Route Configuration
	{
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

		checkIt(expectedDomains, expectedWeightedCluster, v1)
	}

	// created correct 'bookstore-v2' XDS Route Configuration
	{
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

		checkIt(expectedDomains, expectedWeightedCluster, v2)
	}

	{
		//  created correct 'bookstore-apex' XDS Route Configuration
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

		checkIt(expectedDomains, expectedWeightedCluster, apex)
	}
}
