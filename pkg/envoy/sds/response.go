package sds

import (
	"context"
	"fmt"
	"strings"

	xds "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	auth "github.com/envoyproxy/go-control-plane/envoy/api/v2/auth"
	core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	"github.com/golang/protobuf/ptypes"
	"github.com/pkg/errors"

	"github.com/open-service-mesh/osm/pkg/catalog"
	"github.com/open-service-mesh/osm/pkg/certificate"
	"github.com/open-service-mesh/osm/pkg/envoy"
	"github.com/open-service-mesh/osm/pkg/service"
	"github.com/open-service-mesh/osm/pkg/smi"
)

var (
	rootCertPrefix    = fmt.Sprintf("%s%s", envoy.RootCertPrefix, envoy.Separator)
	serviceCertPrefix = fmt.Sprintf("%s%s", envoy.ServiceCertPrefix, envoy.Separator)
)

var validResourceKinds = map[envoy.XDSResourceKind]interface{}{
	envoy.ServiceCertPrefix: nil,
	envoy.RootCertPrefix:    nil,
}

// For each kind of a Resource requested we define a function that handles the response
var taskMakerFuncs = map[envoy.XDSResourceKind]func(resourceName string, serviceForProxy service.NamespacedService, proxyCN certificate.CommonName) (*task, error){
	envoy.ServiceCertPrefix: getServiceCertTask,
	envoy.RootCertPrefix:    getRootCertTask,
}

// NewResponse creates a new Secrets Discovery Response.
func NewResponse(_ context.Context, catalog catalog.MeshCataloger, _ smi.MeshSpec, proxy *envoy.Proxy, request *xds.DiscoveryRequest) (*xds.DiscoveryResponse, error) {
	svc, err := catalog.GetServiceFromEnvoyCertificate(proxy.GetCommonName())
	if err != nil {
		log.Error().Err(err).Msgf("Error looking up Service for Envoy with CN=%q", proxy.GetCommonName())
		return nil, err
	}
	serviceForProxy := *svc

	log.Info().Msgf("Composing SDS Discovery Response for proxy: %s", proxy.GetCommonName())

	cert, err := catalog.GetCertificateForService(serviceForProxy)
	if err != nil {
		log.Error().Err(err).Msgf("Error obtaining a certificate for client %s", proxy.GetCommonName())
		return nil, err
	}

	resp := &xds.DiscoveryResponse{
		TypeUrl: string(envoy.TypeSDS),
	}

	// Iterate over the list of tasks and create response structs to be
	// sent to the proxy that made the discovery request
	for _, task := range getTasks(proxy, request, serviceForProxy) {
		log.Info().Msgf("proxy %s (member of service %s) requested %s", proxy.GetCommonName(), serviceForProxy.String(), task.resourceName)
		secret, err := task.structMaker(cert, task.resourceName)
		if err != nil {
			return nil, errors.Wrapf(err, "error creating cert %s for proxy %s for service %s", task.resourceName, proxy.GetCommonName(), serviceForProxy.String())
		}
		marshalledSecret, err := ptypes.MarshalAny(secret)
		if err != nil {
			return nil, errors.Wrapf(err, "error marshaling cert %s for proxy %s for service %s", task.resourceName, proxy.GetCommonName(), serviceForProxy.String())
		}
		resp.Resources = append(resp.Resources, marshalledSecret)
	}
	return resp, nil
}

func getServiceFromServiceCertificateRequest(resourceName string) (service.NamespacedService, error) {
	// This is a Service Certificate request, which means the resource name begins with "service-cert:"
	// We remove this;  what remains is the namespace and the service separated by a slash:  namespace/service
	slashed := strings.Split(resourceName[len(serviceCertPrefix):], "/")
	if len(slashed) != 2 {
		log.Error().Msgf("Error converting %q into a NamespacedService: expected two strings separated by a slash", resourceName)
		return service.NamespacedService{}, errInvalidResourceRequested
	}

	return service.NamespacedService{
		Namespace: slashed[0],
		Service:   slashed[1],
	}, nil
}

func getServiceFromRootCertificateRequest(resourceName string) (service.NamespacedService, error) {
	// This is a Root Certificate request, which means the resource name begins with "root-cert:"
	// We remove this;  what remains is the namespace and the service separated by a slash:  namespace/service
	slashed := strings.Split(resourceName[len(rootCertPrefix):], "/")
	if len(slashed) != 2 {
		log.Error().Msgf("Error converting %q into a NamespacedService: expected two strings separated by a slash", resourceName)
		return service.NamespacedService{}, errInvalidResourceRequested
	}

	return service.NamespacedService{
		Namespace: slashed[0],
		Service:   slashed[1],
	}, nil
}

func getResourceKindFromRequest(resourceName string) (envoy.XDSResourceKind, error) {
	// The resourceName is of the format "service-cert:namespace/serviceName"
	// The first string before the colon is the resource kind
	// Resource kind could be one of "service-cert" or "root-cert"
	split := strings.Split(resourceName, envoy.Separator)
	if len(split) != 2 {
		log.Error().Msgf("Invalid resourceName requested %q; Expected strings separated by a single colon ':'", resourceName)
		return "", errInvalidResourceName
	}

	kind := envoy.XDSResourceKind(split[0])

	if _, ok := validResourceKinds[kind]; !ok {
		return "", errInvalidResourceKind
	}

	return kind, nil
}

func getServiceCertTask(resourceName string, serviceForProxy service.NamespacedService, proxyCN certificate.CommonName) (*task, error) {
	requestFor, err := getServiceFromServiceCertificateRequest(resourceName)
	if err != nil {
		log.Error().Err(err).Msgf("Error parsing SDS request for resource name: %q", resourceName)
		return nil, err
	}

	if serviceForProxy != requestFor {
		log.Error().Msgf("Proxy %s (service %s) requested service certificate %s; this is not allowed", proxyCN, serviceForProxy, requestFor)
		return nil, errUnauthorizedRequestForServiceFromProxy
	}

	return &task{
		resourceName: resourceName,
		structMaker:  getServiceCertSecret,
	}, nil
}

func getRootCertTask(resourceName string, serviceForProxy service.NamespacedService, proxyCN certificate.CommonName) (*task, error) {
	requestFor, err := getServiceFromRootCertificateRequest(resourceName)
	if err != nil {
		log.Error().Err(err).Msgf("Error parsing SDS request for root certificate: %q", resourceName)
		return nil, err
	}

	if serviceForProxy != requestFor {
		log.Error().Msgf("Proxy %s (service %s) requested root certificate %s; this is not allowed", proxyCN, serviceForProxy, requestFor)
		return nil, errUnauthorizedRequestForRootCertFromProxy
	}

	return &task{
		resourceName: resourceName,
		structMaker:  getRootCert,
	}, nil
}

// getTasks creates a list of tasks (list of structs to be generated) based on the
// proxy that made a discovery request and the discovery request itself
func getTasks(proxy *envoy.Proxy, request *xds.DiscoveryRequest, serviceForProxy service.NamespacedService) []task {
	var tasks []task

	// The Envoy makes a request for a list of resources (aka certificates), which we will send as a response.
	for _, resourceName := range request.ResourceNames {
		// resourceKind could be either "service-cert" or "root-cert"
		resourceKind, err := getResourceKindFromRequest(resourceName)
		if err != nil {
			log.Error().Err(err).Msgf("Invalid resource kind requested: %q", resourceName)
			continue
		}

		taskMakerFunc, ok := taskMakerFuncs[resourceKind]
		if !ok {
			log.Error().Msgf("Request for an unrecognized resource: %s", resourceKind)
			continue
		}

		task, err := taskMakerFunc(resourceName, serviceForProxy, proxy.GetCommonName())
		if err != nil {
			log.Error().Err(err).Msgf("Error creating SDS task for requested resourceName %q for proxy %q", resourceName, proxy.GetCommonName())
			continue
		}

		tasks = append(tasks, *task)
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
	secret := &auth.Secret{
		// The Name field must match the tls_context.common_tls_context.tls_certificate_sds_secret_configs.name
		Name: resourceName,
		Type: &auth.Secret_ValidationContext{
			ValidationContext: &auth.CertificateValidationContext{
				TrustedCa: &core.DataSource{
					Specifier: &core.DataSource_InlineBytes{
						InlineBytes: cert.GetIssuingCA(),
					},
				},
				/*MatchSubjectAltNames: []*envoy_type_matcher.StringMatcher{{
					MatchPattern: &envoy_type_matcher.StringMatcher_Exact{
						Exact: // TODO(draychev) enable this -- see  https://github.com/open-service-mesh/osm/issues/674
					}},
				},
				*/
			},
		},
	}
	return secret, nil
}
