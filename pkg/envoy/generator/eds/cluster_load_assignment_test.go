package eds

import (
	"net"
	"testing"

	xds_core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	xds_endpoint "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	"github.com/google/go-cmp/cmp"
	tassert "github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/openservicemesh/osm/pkg/endpoint"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/service"
)

func TestNewClusterLoadAssignment(t *testing.T) {
	testCases := []struct {
		name      string
		svc       service.MeshService
		endpoints []endpoint.Endpoint
		expected  *xds_endpoint.ClusterLoadAssignment
	}{
		{
			name: "multiple endpoints per cluster within the same locality",
			svc:  service.MeshService{Namespace: "ns1", Name: "bookstore-1", TargetPort: 80},
			endpoints: []endpoint.Endpoint{
				{IP: net.ParseIP("1.1.1.1"), Port: 80},
				{IP: net.ParseIP("2.2.2.2"), Port: 80},
			},
			expected: &xds_endpoint.ClusterLoadAssignment{
				ClusterName: "ns1/bookstore-1|80",
				Endpoints: []*xds_endpoint.LocalityLbEndpoints{
					{
						Locality: &xds_core.Locality{
							Zone: localZone,
						},
						LbEndpoints: []*xds_endpoint.LbEndpoint{
							{
								HostIdentifier: &xds_endpoint.LbEndpoint_Endpoint{
									Endpoint: &xds_endpoint.Endpoint{
										Address: envoy.GetAddress("1.1.1.1", 80),
									},
								},
							},
							{
								HostIdentifier: &xds_endpoint.LbEndpoint_Endpoint{
									Endpoint: &xds_endpoint.Endpoint{
										Address: envoy.GetAddress("2.2.2.2", 80),
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name:      "no endpoints for cluster",
			svc:       service.MeshService{Namespace: "ns1", Name: "bookstore-1", TargetPort: 80},
			endpoints: nil,
			expected: &xds_endpoint.ClusterLoadAssignment{
				ClusterName: "ns1/bookstore-1|80",
				Endpoints: []*xds_endpoint.LocalityLbEndpoints{
					{
						Locality: &xds_core.Locality{
							Zone: localZone,
						},
						LbEndpoints: []*xds_endpoint.LbEndpoint{},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := tassert.New(t)
			actual := newClusterLoadAssignment(tc.svc, tc.endpoints)
			assert.True(cmp.Equal(tc.expected, actual, protocmp.Transform()), cmp.Diff(tc.expected, actual, protocmp.Transform()))
		})
	}
}
