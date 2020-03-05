package sds

import (
	"bytes"
	"context"
	"encoding/pem"
	"fmt"
	"reflect"
	"strings"

	xds "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	auth "github.com/envoyproxy/go-control-plane/envoy/api/v2/auth"
	core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	matcher "github.com/envoyproxy/go-control-plane/envoy/type/matcher"
	"github.com/golang/glog"
	"github.com/golang/protobuf/ptypes"
	"github.com/pkg/errors"

	"github.com/deislabs/smc/pkg/catalog"
	"github.com/deislabs/smc/pkg/certificate"
	"github.com/deislabs/smc/pkg/endpoint"
	"github.com/deislabs/smc/pkg/envoy"
	"github.com/deislabs/smc/pkg/smi"
	"github.com/deislabs/smc/pkg/utils"
)

type empty struct{}

var packageName = utils.GetLastChunkOfSlashed(reflect.TypeOf(empty{}).PkgPath())

// NewResponse creates a new Secrets Discovery Response.
func NewResponse(ctx context.Context, catalog catalog.MeshCataloger, meshSpec smi.MeshSpec, proxy *envoy.Proxy, request *xds.DiscoveryRequest) (*xds.DiscoveryResponse, error) {
	glog.Infof("[%s] Composing SDS Discovery Response for proxy: %s", packageName, proxy.GetCommonName())
	cert, err := catalog.GetCertificateForService(proxy.GetService())
	if err != nil {
		glog.Errorf("[%s] Error obtaining a certificate for client %s: %s", packageName, proxy.GetCommonName(), err)
		return nil, err
	}

	resp := &xds.DiscoveryResponse{
		TypeUrl: string(envoy.TypeSDS),
	}

	tasks := make(map[string]func(cert certificate.Certificater, serviceName endpoint.ServiceName) (*auth.Secret, error))
	for _, resourceName := range request.ResourceNames {
		tasks[fmt.Sprintf("Certificate for %s", resourceName)] = newServiceCertificate
	}
	if len(tasks) == 0 {
		tasks["Root Certificate"] = newRootCertificate
	}

	for description, getSecreteTypeFn := range tasks {
		glog.Infof("[%s] proxy %s (service %s) requested certificates [%s] (%s)", packageName, proxy.GetCommonName(), proxy.GetService(), strings.Join(request.ResourceNames, ","), description)
		secret, err := getSecreteTypeFn(cert, proxy.GetService())
		if err != nil {
			return nil, errors.Wrapf(err, "[%s] error creating new %s for proxy %s for service %s", packageName, description, proxy, proxy.GetService())
		}
		marshalledSecret, err := ptypes.MarshalAny(secret)
		if err != nil {
			return nil, errors.Wrapf(err, "[%s] error marshaling secret for proxy %s for service %s", packageName, proxy, proxy.GetService())
		}
		resp.Resources = append(resp.Resources, marshalledSecret)
	}
	return resp, nil
}

func newServiceCertificate(cert certificate.Certificater, serviceName endpoint.ServiceName) (*auth.Secret, error) {
	secret := &auth.Secret{
		// The Name field must match the tls_context.common_tls_context.tls_certificate_sds_secret_configs.name in the Envoy yaml config
		Name: string(serviceName),
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
	return secret, nil
}

func newRootCertificate(cert certificate.Certificater, serviceName endpoint.ServiceName) (*auth.Secret, error) {
	block := pem.Block{Type: "CERTIFICATE", Bytes: cert.GetRootCertificate().Raw}
	var rootCert bytes.Buffer
	err := pem.Encode(&rootCert, &block)
	if err != nil {
		return nil, errors.Wrap(err, "error PEM encoding certificate")
	}
	secret := &auth.Secret{
		// The Name field must match the tls_context.common_tls_context.tls_certificate_sds_secret_configs.name
		Name: string(serviceName),
		Type: &auth.Secret_ValidationContext{
			ValidationContext: &auth.CertificateValidationContext{
				TrustedCa: &core.DataSource{
					Specifier: &core.DataSource_InlineBytes{
						InlineBytes: rootCert.Bytes(),
					},
				},
				MatchSubjectAltNames: []*matcher.StringMatcher{{
					MatchPattern: &matcher.StringMatcher_Exact{
						Exact: string(serviceName), // TODO(draychev)
					}},
				},
			},
		},
	}
	return secret, nil
}
