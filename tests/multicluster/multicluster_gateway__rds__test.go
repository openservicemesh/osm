package multicluster

import (
	"fmt"
	"testing"

	xds_route "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	"github.com/golang/mock/gomock"
	tassert "github.com/stretchr/testify/assert"

	"github.com/openservicemesh/osm/pkg/envoy/rds"
)

func TestMulticlusterGatewayRouteDiscoveryService(t *testing.T) {
	assert := tassert.New(t)

	// -------------------  SETUP  -------------------
	meshCatalog, proxy, proxyRegistry, mockConfigurator, err := setupMulticlusterGatewayTest(gomock.NewController(t))
	assert.Nil(err, fmt.Sprintf("Error setting up the test: %+v", err))

	// -------------------  LET'S GO  -------------------
	resources, err := rds.NewResponse(meshCatalog, proxy, nil, mockConfigurator, nil, proxyRegistry)
	assert.Nil(err, fmt.Sprintf("rds.NewResponse return unexpected error: %+v", err))
	assert.NotNil(resources, "No RDS resources!")
	assert.Len(resources, 2)

	routeCfg, ok := resources[1].(*xds_route.RouteConfiguration)
	assert.True(ok)
	assert.Equal(routeCfg.Name, "rds-outbound")

	const (
		apexName = "outbound_virtual-host|bookstore-apex.default.local"
		v2Name   = "outbound_virtual-host|bookstore-v2.default.local"
	)

	{
		expectedVHostNames := []string{apexName, v2Name}

		// ---[  Compare with expectations  ]-------
		// created an XDS Route Configuration with 3 outbound virtual hosts
		var actualNames []string
		for _, vHost := range routeCfg.VirtualHosts {
			actualNames = append(actualNames, vHost.Name)
		}

		assert.Equal(len(expectedVHostNames), len(routeCfg.VirtualHosts), fmt.Sprintf("Here are the actual virtual hosts: %+v", actualNames))
		assert.ElementsMatch(actualNames, expectedVHostNames)
	}

	virtualHost := routeCfg.VirtualHosts[0]

	// created correct 'bookstore-v2' XDS Route Configuration
	expectedDomains := []string{
		"bookstore-v2.default",
		"bookstore-v2.default.svc",
		"bookstore-v2.default.svc.cluster",
		"bookstore-v2.default.svc.cluster.local",

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

	assert.Equal(len(expectedDomains), len(virtualHost.Domains))
	assert.ElementsMatch(expectedDomains, virtualHost.Domains)

	assert.Len(virtualHost.Routes, 1)

	expectedClusterCount := len(expectedWeightedCluster.Clusters)
	clusters := virtualHost.Routes[0].GetRoute().GetWeightedClusters()

	assert.Equal(expectedClusterCount, len(clusters.Clusters))
	assert.Equal(expectedWeightedCluster.Clusters[0].String(), clusters.Clusters[0].String())
	assert.Equal(expectedWeightedCluster.TotalWeight.String(), clusters.TotalWeight.String())
}
