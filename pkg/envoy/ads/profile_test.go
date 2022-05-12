package ads

import (
	"fmt"
	"testing"

	xds_cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	xds_discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	"github.com/google/uuid"
	tassert "github.com/stretchr/testify/assert"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/tests"
)

func TestValidateResourcesRequestResponse(t *testing.T) {
	assert := tassert.New(t)
	proxy, err := envoy.NewProxy(certificate.CommonName(fmt.Sprintf("%s.sidecar.foo.bar", uuid.New())), certificate.SerialNumber("123"), tests.NewMockAddress("1.2.3.4"))
	assert.Nil(err)

	testCases := []struct {
		request          *xds_discovery.DiscoveryRequest
		respResources    []types.Resource
		expectDifference int
	}{
		{
			request: &xds_discovery.DiscoveryRequest{
				TypeUrl:       envoy.TypeCDS.String(),
				ResourceNames: []string{"A", "B"},
			},
			respResources: []types.Resource{
				&xds_cluster.Cluster{
					Name: "A",
				},
				&xds_cluster.Cluster{
					Name: "B",
				},
			},
			expectDifference: 0,
		},
		{
			request: &xds_discovery.DiscoveryRequest{
				TypeUrl:       envoy.TypeCDS.String(),
				ResourceNames: []string{"A"},
			},
			respResources: []types.Resource{
				&xds_cluster.Cluster{
					Name: "A",
				},
				&xds_cluster.Cluster{
					Name: "B",
				},
			},
			expectDifference: 0,
		},
		{
			request: &xds_discovery.DiscoveryRequest{
				TypeUrl:       envoy.TypeCDS.String(),
				ResourceNames: []string{},
			},
			respResources: []types.Resource{
				&xds_cluster.Cluster{
					Name: "A",
				},
				&xds_cluster.Cluster{
					Name: "B",
				},
			},
			expectDifference: 0,
		},
		{
			request: &xds_discovery.DiscoveryRequest{
				TypeUrl:       envoy.TypeCDS.String(),
				ResourceNames: []string{"A", "B"},
			},
			respResources: []types.Resource{
				&xds_cluster.Cluster{
					Name: "C",
				},
				&xds_cluster.Cluster{
					Name: "D",
				},
			},
			expectDifference: 2,
		},
	}

	for _, test := range testCases {
		diff := validateRequestResponse(proxy, test.request, test.respResources)
		assert.Equal(test.expectDifference, diff)
	}
}
