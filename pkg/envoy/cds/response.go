package cds

import (
	"fmt"
	"time"

	xds "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	auth "github.com/envoyproxy/go-control-plane/envoy/api/v2/auth"
	v2core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	"github.com/golang/glog"
	"github.com/golang/protobuf/ptypes"

	"github.com/deislabs/smc/pkg/envoy"
	"github.com/deislabs/smc/pkg/log"
)

const (
	sdsClusterName = "sds"
)

func (s *Server) newDiscoveryResponse(proxy envoy.Proxyer) (*xds.DiscoveryResponse, error) {
	glog.Infof("[%s] Composing Cluster Discovery Response for proxy: %s", serverName, proxy.GetCommonName())
	resp := &xds.DiscoveryResponse{
		TypeUrl: typeUrl,
	}

	{
		cluster := getBookstoreCluster()
		glog.V(log.LvlTrace).Infof("[CDS] Constructed ClusterConfiguration: %+v", cluster)
		marshalledClusters, err := ptypes.MarshalAny(cluster)
		if err != nil {
			glog.Errorf("[%s] Failed to marshal cluster for proxy %s: %v", serverName, proxy.GetCommonName(), err)
			return nil, err
		}
		resp.Resources = append(resp.Resources, marshalledClusters)
	}

	{
		edsCluster := getEDS("bookstore.mesh")
		glog.V(log.LvlTrace).Infof("[CDS] Constructed ClusterConfiguration: %+v", edsCluster)
		marshalledClusters, err := ptypes.MarshalAny(edsCluster)
		if err != nil {
			glog.Errorf("[%s] Failed to marshal cluster for proxy %s: %v", serverName, proxy.GetCommonName(), err)
			return nil, err
		}
		resp.Resources = append(resp.Resources, marshalledClusters)
	}

	{
		rdsCluster := getEDS("bookstore.mesh")
		glog.V(log.LvlTrace).Infof("[CDS] Constructed ClusterConfiguration: %+v", rdsCluster)
		marshalledClusters, err := ptypes.MarshalAny(rdsCluster)
		if err != nil {
			glog.Errorf("[%s] Failed to marshal cluster for proxy %s: %v", serverName, proxy.GetCommonName(), err)
			return nil, err
		}
		resp.Resources = append(resp.Resources, marshalledClusters)
	}

	s.lastVersion = s.lastVersion + 1
	s.lastNonce = string(time.Now().Nanosecond())
	resp.Nonce = s.lastNonce
	resp.VersionInfo = fmt.Sprintf("v%d", s.lastVersion)

	glog.V(log.LvlTrace).Infof("[%s] Constructed response: %+v", serverName, resp)

	return resp, nil
}
func getUpstreamTLS() *auth.UpstreamTlsContext {
	return &auth.UpstreamTlsContext{
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
												ClusterName: sdsClusterName,
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
}

func getBookstoreCluster() *xds.Cluster {
	// The name must match the domain being cURLed in the demo
	clusterName := "bookstore.mesh"
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
				TypedConfig: tls,
			},
		},
	}
}
