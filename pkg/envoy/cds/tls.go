package cds

import (
	"github.com/deislabs/smc/pkg/envoy"
	auth "github.com/envoyproxy/go-control-plane/envoy/api/v2/auth"
	core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	"github.com/golang/glog"
	"github.com/golang/protobuf/ptypes"
)

func getTransportSocket() *core.TransportSocket {
	tls, err := ptypes.MarshalAny(getUpstreamTLS())
	if err != nil {
		glog.Error("[CDS] Error marshalling UpstreamTLS: ", err)
		return nil
	}
	return &core.TransportSocket{
		Name:       envoy.TransportSocketTLS,
		ConfigType: &core.TransportSocket_TypedConfig{TypedConfig: tls},
	}
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
					Name: envoy.CertificateName,
					SdsConfig: &core.ConfigSource{
						ConfigSourceSpecifier: &core.ConfigSource_ApiConfigSource{
							ApiConfigSource: &core.ApiConfigSource{
								ApiType: core.ApiConfigSource_GRPC,
								GrpcServices: []*core.GrpcService{
									{
										TargetSpecifier: &core.GrpcService_EnvoyGrpc_{
											EnvoyGrpc: &core.GrpcService_EnvoyGrpc{
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
