package cds

import (
	"fmt"
	"time"

	xds "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/auth"
	v2core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	"github.com/gogo/protobuf/types"
	"github.com/golang/glog"

	"github.com/deislabs/smc/pkg/envoy"
	"github.com/deislabs/smc/pkg/logging"
)

func (s *Server) newDiscoveryResponse(proxy envoy.Proxyer) (*xds.DiscoveryResponse, error) {
	glog.Infof("[%s] Composing Cluster Discovery Response for proxy: %s", serverName, proxy.GetCommonName())
	resp := &xds.DiscoveryResponse{
		TypeUrl: typeUrl,
	}

	// The name must match the domain being cURLed in the demo
	clusterName := "bookstore.mesh"
	connTimeout := 10 * time.Second

	upstreamTLS := &auth.UpstreamTlsContext{
		AllowRenegotiation: true,
		CommonTlsContext: &auth.CommonTlsContext{
			TlsParams:       nil,
			TlsCertificates: nil,
			TlsCertificateSdsSecretConfigs: []*auth.SdsSecretConfig{
				{
					// The Name field must match the auth.Secret.Name from the SDS response
					Name: "server_cert",
					SdsConfig: &v2core.ConfigSource{
						ConfigSourceSpecifier: &v2core.ConfigSource_ApiConfigSource{
							ApiConfigSource: &v2core.ApiConfigSource{
								ApiType: v2core.ApiConfigSource_GRPC,
								GrpcServices: []*v2core.GrpcService{
									{
										TargetSpecifier: &v2core.GrpcService_EnvoyGrpc_{
											EnvoyGrpc: &v2core.GrpcService_EnvoyGrpc{
												// This must match the hard-coded SDS cluster name in the bootstrap config
												ClusterName: "sds",
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	tlsAny, err := types.MarshalAny(upstreamTLS)

	if err != nil {
		return nil, err
	}

	cluster := &xds.Cluster{
		// The name must match the domain being cURLed in the demo
		Name:           clusterName,
		AltStatName:    clusterName,
		ConnectTimeout: &connTimeout,
		LbPolicy:       xds.Cluster_ROUND_ROBIN,
		RespectDnsTtl: true,
		DrainConnectionsOnHostRemoval: true,
		ClusterDiscoveryType: &xds.Cluster_Type{
			Type: xds.Cluster_EDS,
		},
		EdsClusterConfig: &xds.Cluster_EdsClusterConfig{
			ServiceName: clusterName,
			EdsConfig: &v2core.ConfigSource{
				ConfigSourceSpecifier: &v2core.ConfigSource_ApiConfigSource{
					ApiConfigSource: &v2core.ApiConfigSource{
						ApiType: v2core.ApiConfigSource_GRPC,
						GrpcServices: []*v2core.GrpcService{
							{
								TargetSpecifier: &v2core.GrpcService_EnvoyGrpc_{
									EnvoyGrpc: &v2core.GrpcService_EnvoyGrpc{
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
		TransportSocket: &v2core.TransportSocket{
			Name: "envoy.transport_sockets.tls",
			ConfigType: &v2core.TransportSocket_TypedConfig{
				TypedConfig: tlsAny,
			},
		},
	}
	glog.V(log.LvlTrace).Infof("[CDS] Constructed ClusterConfiguratio: %+v", cluster)
	marshalledClusters, err := types.MarshalAny(cluster)
	if err != nil {
		glog.Errorf("[%s] Failed to marshal cluster for proxy %s: %v", serverName, proxy.GetCommonName(), err)
		return nil, err
	}
	resp.Resources = append(resp.Resources, marshalledClusters)

	s.lastVersion = s.lastVersion + 1
	s.lastNonce = string(time.Now().Nanosecond())
	resp.Nonce = s.lastNonce
	resp.VersionInfo = fmt.Sprintf("v%d", s.lastVersion)

	glog.V(log.LvlTrace).Infof("[%s] Constructed response: %+v", serverName, resp)

	return resp, nil
}
