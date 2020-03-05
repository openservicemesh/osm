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

type tuple struct {
	certMaker    func(certificate.Certificater, string) (*auth.Secret, error)
	resourceName string
}

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

	rootCertPrefix := fmt.Sprintf("%s%s", envoy.RootCertPrefix, envoy.Separator)
	serviceCertPrefix := fmt.Sprintf("%s%s", envoy.ServiceCertPrefix, envoy.Separator)

	var tasks []tuple
	for _, resourceName := range request.ResourceNames {
		if strings.HasPrefix(resourceName, serviceCertPrefix) {
			requestFor := endpoint.ServiceName(resourceName[len(serviceCertPrefix):])
			if proxy.GetService() != requestFor {
				glog.Errorf("[%s] Proxy %s (service %s) requested service certificate %s; this is not allowed", packageName, proxy.GetCommonName(), proxy.GetService(), requestFor)
				continue
			}
			tasks = append(tasks, tuple{
				resourceName: resourceName,
				certMaker:    getServiceCert,
			})
		} else if strings.HasPrefix(resourceName, rootCertPrefix) {
			requestFor := endpoint.ServiceName(resourceName[len(rootCertPrefix):])
			if proxy.GetService() != requestFor {
				glog.Errorf("[%s] Proxy %s (service %s) requested root certificate %s; this is not allowed", packageName, proxy.GetCommonName(), proxy.GetService(), requestFor)
				continue
			}
			tasks = append(tasks, tuple{
				resourceName: resourceName,
				certMaker:    getRootCert,
			})
		} else {
			glog.Errorf("[%s] Request for an unrecognized resource: %s", packageName, resourceName)
			continue
		}
	}

	for _, tpl := range tasks {
		glog.Infof("[%s] proxy %s (member of service %s) requested %s", packageName, proxy.GetCommonName(), proxy.GetService(), tpl.resourceName)
		secret, err := tpl.certMaker(cert, tpl.resourceName)
		if err != nil {
			return nil, errors.Wrapf(err, "[%s] error creating cert %s for proxy %s for service %s", packageName, tpl.resourceName, proxy.GetCommonName(), proxy.GetService())
		}
		marshalledSecret, err := ptypes.MarshalAny(secret)
		if err != nil {
			return nil, errors.Wrapf(err, "[%s] error marshaling cert %s for proxy %s for service %s", packageName, tpl.resourceName, proxy.GetCommonName(), proxy.GetService())
		}
		resp.Resources = append(resp.Resources, marshalledSecret)
	}
	return resp, nil
}

func getServiceCert(cert certificate.Certificater, name string) (*auth.Secret, error) {
	secret := &auth.Secret{
		// The Name field must match the tls_context.common_tls_context.tls_certificate_sds_secret_configs.name in the Envoy yaml config
		Name: name,
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

func getRootCert(cert certificate.Certificater, resourceName string) (*auth.Secret, error) {
	block := pem.Block{Type: "CERTIFICATE", Bytes: cert.GetRootCertificate().Raw}
	var rootCert bytes.Buffer
	err := pem.Encode(&rootCert, &block)
	if err != nil {
		return nil, errors.Wrap(err, "error PEM encoding certificate")
	}
	secret := &auth.Secret{
		// The Name field must match the tls_context.common_tls_context.tls_certificate_sds_secret_configs.name
		Name: resourceName,
		Type: &auth.Secret_ValidationContext{
			ValidationContext: &auth.CertificateValidationContext{
				TrustedCa: &core.DataSource{
					Specifier: &core.DataSource_InlineBytes{
						InlineBytes: rootCert.Bytes(),
					},
				},
				/*MatchSubjectAltNames: []*envoy_type_matcher.StringMatcher{{
					MatchPattern: &envoy_type_matcher.StringMatcher_Exact{
						Exact: cert.GetName(), // TODO(draychev): flesh out
					}},
				},*/
			},
		},
	}
	return secret, nil
}
