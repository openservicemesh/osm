package cds

import (
	"time"

	"github.com/deislabs/smc/pkg/envoy"
	xds "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	endpoint "github.com/envoyproxy/go-control-plane/envoy/api/v2/endpoint"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/wrappers"
)

func getServiceClusterLocal(clusterName string) *xds.Cluster {
	return &xds.Cluster{
		// The name must match the domain being cURLed in the demo
		Name:                          clusterName,
		AltStatName:                   clusterName,
		ConnectTimeout:                ptypes.DurationProto(1 * time.Second),
		LbPolicy:                      xds.Cluster_ROUND_ROBIN,
		RespectDnsTtl:                 true,
		DrainConnectionsOnHostRemoval: true,
		ClusterDiscoveryType: &xds.Cluster_Type{
			Type: xds.Cluster_STRICT_DNS,
		},
		DnsLookupFamily: xds.Cluster_V4_ONLY,
		LoadAssignment: &xds.ClusterLoadAssignment{
			// NOTE: results.ServiceName is the top level service that is cURLed.
			ClusterName: clusterName,
			Endpoints: []*endpoint.LocalityLbEndpoints{
				{
					Locality: &core.Locality{
						Zone: "zone",
					},
					LbEndpoints: []*endpoint.LbEndpoint{{
						HostIdentifier: &endpoint.LbEndpoint_Endpoint{
							Endpoint: &endpoint.Endpoint{
								Address: envoy.GetAddress("0.0.0.0", 80),
							},
						},
						LoadBalancingWeight: &wrappers.UInt32Value{
							Value: 100,
						},
					}},
				},
			},
		},
	}
}
