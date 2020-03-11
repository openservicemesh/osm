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

	"github.com/open-service-mesh/osm/pkg/catalog"
	"github.com/open-service-mesh/osm/pkg/certificate"
	"github.com/open-service-mesh/osm/pkg/endpoint"
	"github.com/open-service-mesh/osm/pkg/envoy"
	"github.com/open-service-mesh/osm/pkg/smi"
	"github.com/open-service-mesh/osm/pkg/utils"
)

type empty struct{}

var packageName = utils.GetLastChunkOfSlashed(reflect.TypeOf(empty{}).PkgPath())

type task struct {
	structMaker  func(certificate.Certificater, string) (*auth.Secret, error)
	resourceName string
}

var (
	rootCertPrefix    = fmt.Sprintf("%s%s", envoy.RootCertPrefix, envoy.Separator)
	serviceCertPrefix = fmt.Sprintf("%s%s", envoy.ServiceCertPrefix, envoy.Separator)
)

// NewResponse creates a new Secrets Discovery Response.
func NewResponse(ctx context.Context, catalog catalog.MeshCataloger, meshSpec smi.MeshSpec, proxy *envoy.Proxy, request *xds.DiscoveryRequest) (*xds.DiscoveryResponse, error) {
	glog.Infof("[%s] Composing SDS Discovery Response for proxy: %s", packageName, proxy.GetCommonName())
	proxyServiceName := proxy.GetService()
	cert, err := catalog.GetCertificateForService(proxyServiceName)
	if err != nil {
		glog.Errorf("[%s] Error obtaining a certificate for client %s: %s", packageName, proxy.GetCommonName(), err)
		return nil, err
	}

	resp := &xds.DiscoveryResponse{
		TypeUrl: string(envoy.TypeSDS),
	}

	// Iterate over the list of tasks and create response structs to be
	// sent to the proxy that made the discovery request
	for _, task := range getTasks(proxy, request) {
		glog.Infof("[%s] proxy %s (member of service %s) requested %s", packageName, proxy.GetCommonName(), proxyServiceName.String(), task.resourceName)
		secret, err := task.structMaker(cert, task.resourceName)
		if err != nil {
			return nil, errors.Wrapf(err, "[%s] error creating cert %s for proxy %s for service %s", packageName, task.resourceName, proxy.GetCommonName(), proxyServiceName.String())
		}
		marshalledSecret, err := ptypes.MarshalAny(secret)
		if err != nil {
			return nil, errors.Wrapf(err, "[%s] error marshaling cert %s for proxy %s for service %s", packageName, task.resourceName, proxy.GetCommonName(), proxyServiceName.String())
		}
		resp.Resources = append(resp.Resources, marshalledSecret)
	}
	return resp, nil
}

// getTasks creates a list of tasks (list of structs to be generated) based on the
// proxy that made a discovery request and the discovery request itself
func getTasks(proxy *envoy.Proxy, request *xds.DiscoveryRequest) []task {
	var tasks []task
	proxyServiceName := proxy.GetService()

	// the proxy may have made a request with a number of resources (certificates) expected to be sent back
	for _, resourceName := range request.ResourceNames {
		if strings.HasPrefix(resourceName, serviceCertPrefix) {
			// this is a request for a service certificate
			requestFor := endpoint.ServiceName(resourceName[len(serviceCertPrefix):])
			if endpoint.ServiceName(proxyServiceName.String()) != requestFor {
				glog.Errorf("[%s] Proxy %s (service %s) requested service certificate %s; this is not allowed", packageName, proxy.GetCommonName(), proxy.GetService(), requestFor)
				continue
			}
			tasks = append(tasks, task{
				resourceName: resourceName,
				structMaker:  getServiceCertSecret,
			})
		} else if strings.HasPrefix(resourceName, rootCertPrefix) {
			// this is a request for a root certificate
			// proxies need this to verify other proxies certificates
			requestFor := getServiceName(resourceName, envoy.RootCertPrefix)
			if endpoint.ServiceName(proxyServiceName.String()) != requestFor {
				glog.Errorf("[%s] Proxy %s (service %s) requested root certificate %s; this is not allowed", packageName, proxy.GetCommonName(), proxy.GetService(), requestFor)
				continue
			}
			tasks = append(tasks, task{
				resourceName: resourceName,
				structMaker:  getRootCert,
			})
		} else {
			glog.Errorf("[%s] Request for an unrecognized resource: %s", packageName, resourceName)
			continue
		}
	}
	return tasks
}

// getServiceCertSecret creates the struct with certificates for the service, which the
// connected Envoy proxy belongs to.
func getServiceCertSecret(cert certificate.Certificater, name string) (*auth.Secret, error) {
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
						Exact: // TODO(draychev)
					}},
				},
				*/
			},
		},
	}
	return secret, nil
}

func getServiceName(resourceName string, prefix string) endpoint.ServiceName {
	rootCertPrefix := fmt.Sprintf("%s%s", prefix, envoy.Separator)
	return endpoint.ServiceName(resourceName[len(rootCertPrefix):])
}
