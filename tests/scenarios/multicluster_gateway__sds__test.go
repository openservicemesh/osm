package scenarios

import (
	"fmt"
	"sort"
	"testing"

	xds_auth "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	"github.com/golang/mock/gomock"
	tassert "github.com/stretchr/testify/assert"

	"github.com/openservicemesh/osm/pkg/envoy/sds"
)

type XDSSecret []*xds_auth.Secret

// Satisfy sort.Interface
func (c XDSSecret) Len() int           { return len(c) }
func (c XDSSecret) Less(i, j int) bool { return c[i].Name < c[j].Name }
func (c XDSSecret) Swap(i, j int)      { c[i], c[j] = c[j], c[i] }

func TestMulticlusterGatewaySecretDiscoveryService(t *testing.T) {
	assert := tassert.New(t)

	// -------------------  SETUP  -------------------
	meshCatalog, proxy, proxyRegistry, mockConfigurator, err := setupMulticlusterGatewayTest(gomock.NewController(t))
	assert.Nil(err, fmt.Sprintf("Error setting up the test: %+v", err))

	// -------------------  TEST  -------------------
	resources, err := sds.NewResponse(meshCatalog, proxy, nil, mockConfigurator, nil, proxyRegistry)
	assert.Nil(err, fmt.Sprintf("sds.NewResponse return unexpected error: %+v", err))
	assert.NotNil(resources, "No SDS resources!")
	assert.Len(resources, 4)

	var secrets XDSSecret
	for _, xdsResource := range resources {
		cluster, ok := xdsResource.(*xds_auth.Secret)
		assert.True(ok)
		secrets = append(secrets, cluster)
	}

	sort.Sort(secrets)

	assert.Equal("default/bookbuyer", secrets[0].Name)
	assert.Equal("default/bookstore-apex", secrets[1].Name)
	assert.Equal("default/bookstore-v1", secrets[2].Name)
	assert.Equal("default/bookstore-v2", secrets[3].Name)
}
