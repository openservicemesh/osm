package utils

import (
	"testing"

	xds_cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	xds_core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	xds_endpoint "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	xds_transport_sockets "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	"github.com/golang/protobuf/ptypes/wrappers"
	tassert "github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func TestProtoToYAML(t *testing.T) {
	testCases := []struct {
		name         string
		proto        protoreflect.ProtoMessage
		expectedYAML string
	}{
		{
			name: "XDS cluster proto",
			proto: &xds_cluster.Cluster{
				TransportSocketMatches: nil,
				Name:                   "foo",
				AltStatName:            "foo",
				ClusterDiscoveryType:   &xds_cluster.Cluster_Type{Type: xds_cluster.Cluster_STATIC},
				EdsClusterConfig:       nil,
				LoadAssignment: &xds_endpoint.ClusterLoadAssignment{
					ClusterName: "foo",
					Endpoints: []*xds_endpoint.LocalityLbEndpoints{
						{
							Locality: nil,
							LbEndpoints: []*xds_endpoint.LbEndpoint{{
								HostIdentifier: &xds_endpoint.LbEndpoint_Endpoint{
									Endpoint: &xds_endpoint.Endpoint{
										Address: &xds_core.Address{
											Address: &xds_core.Address_SocketAddress{
												SocketAddress: &xds_core.SocketAddress{
													Protocol: xds_core.SocketAddress_TCP,
													Address:  "127.0.0.1",
													PortSpecifier: &xds_core.SocketAddress_PortValue{
														PortValue: 80,
													},
												},
											},
										},
									},
								},
								LoadBalancingWeight: &wrappers.UInt32Value{
									Value: 100,
								},
							}},
						},
					},
				},
			},
			expectedYAML: `alt_stat_name: foo
load_assignment:
  cluster_name: foo
  endpoints:
  - lb_endpoints:
    - endpoint:
        address:
          socket_address:
            address: 127.0.0.1
            port_value: 80
      load_balancing_weight: 100
name: foo
type: STATIC
`,
		},
		{
			name: "TLS params proto",
			proto: &xds_transport_sockets.TlsParameters{
				TlsMinimumProtocolVersion: xds_transport_sockets.TlsParameters_TLSv1_2,
				TlsMaximumProtocolVersion: xds_transport_sockets.TlsParameters_TLSv1_3,
			},
			expectedYAML: `tls_maximum_protocol_version: TLSv1_3
tls_minimum_protocol_version: TLSv1_2
`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := tassert.New(t)

			actual, err := ProtoToYAML(tc.proto)
			assert.Nil(err)
			assert.Equal(tc.expectedYAML, string(actual))
		})
	}
}
