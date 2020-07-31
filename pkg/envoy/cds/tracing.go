package cds

import (
	xds "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	envoyEndpoint "github.com/envoyproxy/go-control-plane/envoy/api/v2/endpoint"
	"github.com/golang/protobuf/ptypes"

	"github.com/open-service-mesh/osm/pkg/constants"
	"github.com/open-service-mesh/osm/pkg/envoy"
)

func getZipkinCluster(zipkinHostname string) xds.Cluster {
	return xds.Cluster{
		Name:           constants.EnvoyZipkinCluster,
		AltStatName:    constants.EnvoyZipkinCluster,
		ConnectTimeout: ptypes.DurationProto(clusterConnectTimeout),
		ClusterDiscoveryType: &xds.Cluster_Type{
			Type: xds.Cluster_LOGICAL_DNS,
		},
		LbPolicy: xds.Cluster_ROUND_ROBIN,
		LoadAssignment: &xds.ClusterLoadAssignment{
			ClusterName: constants.EnvoyZipkinCluster,
			Endpoints: []*envoyEndpoint.LocalityLbEndpoints{
				{
					LbEndpoints: []*envoyEndpoint.LbEndpoint{{
						HostIdentifier: &envoyEndpoint.LbEndpoint_Endpoint{
							Endpoint: &envoyEndpoint.Endpoint{
								Address: envoy.GetAddress(zipkinHostname, constants.EnvoyZipkinPort),
							},
						},
					}},
				},
			},
		},
	}
}
