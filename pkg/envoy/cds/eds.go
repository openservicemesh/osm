package cds

import (
	envoyapiv2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	"github.com/golang/protobuf/ptypes/duration"
)

const (
	edsClusterName = "eds"
)

func getEDS(serviceName string) *envoyapiv2.Cluster {

	return &envoyapiv2.Cluster{
		Name:                          serviceName,
		LbPolicy:                      envoyapiv2.Cluster_ROUND_ROBIN,
		RespectDnsTtl:                 true,
		DrainConnectionsOnHostRemoval: true,
		ConnectTimeout: &duration.Duration{
			Seconds: 10,
		},
		ClusterDiscoveryType: &envoyapiv2.Cluster_Type{
			Type: envoyapiv2.Cluster_EDS,
		},
		EdsClusterConfig: &envoyapiv2.Cluster_EdsClusterConfig{
			ServiceName: serviceName,
			EdsConfig: &core.ConfigSource{
				ConfigSourceSpecifier: &core.ConfigSource_ApiConfigSource{
					ApiConfigSource: &core.ApiConfigSource{
						ApiType: core.ApiConfigSource_GRPC,
						GrpcServices: []*core.GrpcService{
							{
								TargetSpecifier: &core.GrpcService_EnvoyGrpc_{
									EnvoyGrpc: &core.GrpcService_EnvoyGrpc{
										ClusterName: edsClusterName,
									},
								},
							},
						},
					},
				},
			},
		},
	}
}
