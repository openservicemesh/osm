package ads

import (
	"testing"
	"time"

	xds_cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	xds_discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	"github.com/google/uuid"
	tassert "github.com/stretchr/testify/assert"

	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/tests"
)

func TestGetXDSLog(t *testing.T) {
	assert := tassert.New(t)

	testXDSLog := make(map[string]map[envoy.TypeURI][]time.Time)
	testXDSLog["abra"] = make(map[envoy.TypeURI][]time.Time)
	testXDSLog["abra"]["cadabra"] = []time.Time{time.Now()}

	s := Server{
		xdsLog: testXDSLog,
	}

	res := s.GetXDSLog()
	assert.Equal(res, testXDSLog)
}

func TestValidateResourcesRequestResponse(t *testing.T) {
	assert := tassert.New(t)
	proxy := envoy.NewProxy(envoy.KindSidecar, uuid.New(), identity.New("foo", "bar"), tests.NewMockAddress("1.2.3.4"), 1)

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
