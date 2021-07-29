package scenarios

import (
	"fmt"
	"sort"
	"testing"

	xds_cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	"github.com/golang/mock/gomock"
	tassert "github.com/stretchr/testify/assert"

	"github.com/openservicemesh/osm/pkg/envoy/cds"
)

type XDSClusters []*xds_cluster.Cluster

// Satisfy sort.Interface
func (c XDSClusters) Len() int           { return len(c) }
func (c XDSClusters) Less(i, j int) bool { return c[i].Name < c[j].Name }
func (c XDSClusters) Swap(i, j int)      { c[i], c[j] = c[j], c[i] }

func TestMulticlusterGatewayClusterDiscoveryService(t *testing.T) {
	assert := tassert.New(t)

	// -------------------  SETUP  -------------------
	meshCatalog, proxy, proxyRegistry, mockConfigurator, err := setupMulticlusterGatewayTest(gomock.NewController(t))
	assert.Nil(err, fmt.Sprintf("Error setting up the test: %+v", err))

	// -------------------  TEST  -------------------
	resources, err := cds.NewResponse(meshCatalog, proxy, nil, mockConfigurator, nil, proxyRegistry)
	assert.Nil(err, fmt.Sprintf("cds.NewResponse return unexpected error: %+v", err))
	assert.NotNil(resources, "No CDS resources!")
	assert.Len(resources, 4)

	var clusters XDSClusters
	for _, xdsResource := range resources {
		cluster, ok := xdsResource.(*xds_cluster.Cluster)
		assert.True(ok)
		clusters = append(clusters, cluster)
	}

	sort.Sort(clusters)

	assert.Equal("bookbuyer.default.svc.cluster.local", clusters[0].Name)
	assert.Equal("bookstore-apex.default.svc.cluster.local", clusters[1].Name)
	assert.Equal("bookstore-v1.default.svc.cluster.local", clusters[2].Name)
	assert.Equal("bookstore-v2.default.svc.cluster.local", clusters[3].Name)
}
