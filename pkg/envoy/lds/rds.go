package lds

import (
	core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
)

const (
	rdsClusterName = "rds"
)

func getRDSSource() *core.ConfigSource {
	service := &core.GrpcService{
		TargetSpecifier: &core.GrpcService_EnvoyGrpc_{
			EnvoyGrpc: &core.GrpcService_EnvoyGrpc{
				ClusterName: rdsClusterName,
			},
		},
	}

	rds := &core.ConfigSource{
		ConfigSourceSpecifier: &core.ConfigSource_ApiConfigSource{
			ApiConfigSource: &core.ApiConfigSource{
				ApiType:                   core.ApiConfigSource_GRPC,
				GrpcServices:              []*core.GrpcService{service},
				SetNodeOnFirstMessageOnly: true,
			},
		},
	}
	return rds
}
