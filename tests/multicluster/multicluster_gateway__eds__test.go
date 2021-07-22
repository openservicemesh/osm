package multicluster

import (
	"fmt"
	"testing"

	"github.com/openservicemesh/osm/pkg/envoy/eds"

	xds_endpoint "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	"github.com/golang/mock/gomock"
	tassert "github.com/stretchr/testify/assert"
)

func TestMulticlusterGatewayEndpointDiscoveryService(t *testing.T) {
	assert := tassert.New(t)

	// -------------------  SETUP  -------------------
	meshCatalog, proxy, proxyRegistry, mockConfigurator, err := setupMulticlusterGatewayTest(gomock.NewController(t))
	assert.Nil(err, fmt.Sprintf("Error setting up the test: %+v", err))

	// -------------------  LET'S GO  -------------------
	resources, err := eds.NewResponse(meshCatalog, proxy, nil, mockConfigurator, nil, proxyRegistry)
	assert.Nil(err, fmt.Sprintf("eds.NewResponse return unexpected error: %+v", err))
	assert.NotNil(resources, "No EDS resources!")
	assert.Len(resources, 1)

	var clusterLoadAssignments []*xds_endpoint.ClusterLoadAssignment
	for _, xdsResource := range resources {
		clusterLoadAssignment, ok := xdsResource.(*xds_endpoint.ClusterLoadAssignment)
		assert.True(ok)
		clusterLoadAssignments = append(clusterLoadAssignments, clusterLoadAssignment)
	}

	cla := clusterLoadAssignments[0]

	assert.Equal("default/bookstore-v2/local", cla.ClusterName)
	assert.Len(cla.Endpoints, 1)
	assert.Len(cla.NamedEndpoints, 0)
	assert.Len(cla.Endpoints[0].LbEndpoints, 2) // TODO(draychev): WHY 2??
	assert.Equal(cla.Endpoints[0].Locality.Zone, "zone")

	socketAddress0 := cla.Endpoints[0].LbEndpoints[0].GetEndpoint().Address.GetSocketAddress()
	assert.Equal("8.8.8.8", socketAddress0.Address)
	assert.Equal(uint32(8888), socketAddress0.GetPortValue())

	socketAddress1 := cla.Endpoints[0].LbEndpoints[1].GetEndpoint().Address.GetSocketAddress()
	assert.Equal("8.8.8.8", socketAddress1.Address)
	assert.Equal(uint32(8888), socketAddress1.GetPortValue())
}
