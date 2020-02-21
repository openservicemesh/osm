package sds

import (
	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	auth "github.com/envoyproxy/go-control-plane/envoy/api/v2/auth"
	core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	"github.com/golang/glog"
	"github.com/golang/protobuf/ptypes"

	"github.com/deislabs/smc/pkg/envoy"
)

const (
	serverName = "SDS"
)

// NewDiscoveryResponse creates a new Secrets Discovery Response.
func (s *Server) NewDiscoveryResponse(proxy *envoy.Proxy) (*v2.DiscoveryResponse, error) {
	glog.Infof("[%s] Composing SDS Discovery Response for proxy: %s", serverName, proxy.GetCommonName())
	cert, err := s.catalog.GetCertificateForService(proxy.GetService())
	if err != nil {
		glog.Errorf("[%s] Error obtaining a certificate for client %s: %s", serverName, proxy.GetCommonName(), err)
		return nil, err
	}

	resp := &v2.DiscoveryResponse{
		TypeUrl: string(envoy.TypeSDS),
	}

	services := []string{
		envoy.CertificateName,
		"bookstore.mesh",
		"bookstore-1",
		"bookstore-2",
	}

	for _, svc := range services {
		secret := &auth.Secret{
			// The Name field must match the tls_context.common_tls_context.tls_certificate_sds_secret_configs.name in the Envoy yaml config
			Name: svc, // cert.GetName(),
			Type: &auth.Secret_TlsCertificate{
				TlsCertificate: &auth.TlsCertificate{
					CertificateChain: &core.DataSource{
						Specifier: &core.DataSource_InlineBytes{
							InlineBytes: cert.GetCertificateChain(),
						},
					},
					PrivateKey: &core.DataSource{
						Specifier: &core.DataSource_InlineBytes{
							InlineBytes: cert.GetPrivateKey(),
						},
					},
				},
			},
		}
		marshalledSecret, err := ptypes.MarshalAny(secret)
		if err != nil {
			glog.Errorf("[%s] Failed to marshal secret for proxy %s: %v", serverName, proxy.GetCommonName(), err)
			return nil, err
		}
		resp.Resources = append(resp.Resources, marshalledSecret)
	}
	return resp, nil
}
