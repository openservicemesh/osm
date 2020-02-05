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

func buildTlsCertChain() *auth.TlsCertificate {
	return &auth.TlsCertificate{
		CertificateChain: &core.DataSource{
			Specifier: &core.DataSource_Filename{
				Filename: "/etc/ssl/certs/cert.pem",
			},
		},
		PrivateKey: &core.DataSource{
			Specifier: &core.DataSource_Filename{
				Filename: "/etc/ssl/certs/key.pem",
			},
		},
	}
}

func getUpstreamTLS() *auth.UpstreamTlsContext {
	return &auth.UpstreamTlsContext{
		AllowRenegotiation: true,
		CommonTlsContext: &auth.CommonTlsContext{
			TlsParams: &auth.TlsParameters{
				TlsMinimumProtocolVersion: 3,
				TlsMaximumProtocolVersion: 4,
			},
			TlsCertificates: []*auth.TlsCertificate{
				buildTlsCertChain(),
			},
		},
	}
}
