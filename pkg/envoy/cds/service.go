package cds

import (
	"time"

	xds "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	"github.com/golang/glog"
	"github.com/golang/protobuf/ptypes"
)

func getServiceCluster(clusterName string) *xds.Cluster {
	connTimeout := ptypes.DurationProto(10 * time.Second)

	tls, err := ptypes.MarshalAny(getUpstreamTLS())

	if err != nil {
		glog.Error("[CDS] Could not marshal the Upstream TLS: ", err)
		return nil
	}

	return &xds.Cluster{
		// The name must match the domain being cURLed in the demo
		Name:                          clusterName,
		AltStatName:                   clusterName,
		ConnectTimeout:                connTimeout,
		LbPolicy:                      xds.Cluster_ROUND_ROBIN,
		RespectDnsTtl:                 true,
		DrainConnectionsOnHostRemoval: true,
		ClusterDiscoveryType: &xds.Cluster_Type{
			Type: xds.Cluster_EDS,
		},
		EdsClusterConfig: &xds.Cluster_EdsClusterConfig{
			ServiceName: clusterName,
			EdsConfig: &core.ConfigSource{
				ConfigSourceSpecifier: &core.ConfigSource_ApiConfigSource{
					ApiConfigSource: &core.ApiConfigSource{
						ApiType: core.ApiConfigSource_GRPC,
						GrpcServices: []*core.GrpcService{
							{
								TargetSpecifier: &core.GrpcService_EnvoyGrpc_{
									EnvoyGrpc: &core.GrpcService_EnvoyGrpc{
										// This must match the hard-coded EDS cluster name in the bootstrap config
										ClusterName: "eds",
									},
								},
							},
						},
					},
				},
			},
		},
		TransportSocket: &core.TransportSocket{
			Name: "envoy.transport_sockets.tls",
			ConfigType: &core.TransportSocket_TypedConfig{
				TypedConfig: tls,
			},
		},
	}
}
