package sds

import (
	"context"
	"reflect"

	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	auth "github.com/envoyproxy/go-control-plane/envoy/api/v2/auth"
	core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	"github.com/golang/glog"
	"github.com/golang/protobuf/ptypes"

	"github.com/deislabs/smc/pkg/catalog"
	"github.com/deislabs/smc/pkg/certificate"
	"github.com/deislabs/smc/pkg/envoy"
	"github.com/deislabs/smc/pkg/smi"
	"github.com/deislabs/smc/pkg/utils"
)

type empty struct{}

var packageName = utils.GetLastChunkOfSlashed(reflect.TypeOf(empty{}).PkgPath())

// NewResponse creates a new Secrets Discovery Response.
func NewResponse(ctx context.Context, catalog catalog.MeshCataloger, meshSpec smi.MeshSpec, proxy *envoy.Proxy) (*v2.DiscoveryResponse, error) {
	glog.Infof("[%s] Composing SDS Discovery Response for proxy: %s", packageName, proxy.GetCommonName())
	cert, err := catalog.GetCertificateForService(proxy.GetService())
	if err != nil {
		glog.Errorf("[%s] Error obtaining a certificate for client %s: %s", packageName, proxy.GetCommonName(), err)
		return nil, err
	}

	resp := &v2.DiscoveryResponse{
		TypeUrl: string(envoy.TypeSDS),
	}

	serverSecret := newSecret(cert, string(proxy.GetService()))
	marshalledSecret, err := ptypes.MarshalAny(serverSecret)
	if err != nil {
		glog.Errorf("[%s] Failed to marshal secret for proxy %s: %v", packageName, proxy.GetCommonName(), err)
		return nil, err
	}
	resp.Resources = append(resp.Resources, marshalledSecret)
	return resp, nil
}

func newSecret(cert certificate.Certificater, serviceName string) *auth.Secret {
	secret := &auth.Secret{
		// The Name field must match the tls_context.common_tls_context.tls_certificate_sds_secret_configs.name in the Envoy yaml config
		Name: serviceName, // cert.GetName(),
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
	return secret
}
