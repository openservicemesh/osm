package sds

import (
	"context"
	"fmt"
	"strings"

	xds "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	auth "github.com/envoyproxy/go-control-plane/envoy/api/v2/auth"
	core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	envoy_type_matcher "github.com/envoyproxy/go-control-plane/envoy/type/matcher"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"

	"github.com/open-service-mesh/osm/pkg/catalog"
	"github.com/open-service-mesh/osm/pkg/certificate"
	"github.com/open-service-mesh/osm/pkg/envoy"
	"github.com/open-service-mesh/osm/pkg/service"
	"github.com/open-service-mesh/osm/pkg/smi"
)

var (
	rootCertStrictPrefix  = fmt.Sprintf("%s%s", envoy.RootCertTypeForMTLS, envoy.Separator)
	rootCertRelaxedPrefix = fmt.Sprintf("%s%s", envoy.RootCertTypeForHTTPS, envoy.Separator)
	serviceCertPrefix     = fmt.Sprintf("%s%s", envoy.ServiceCertType, envoy.Separator)
)

var validResourceTypes = map[envoy.SDSCertType]interface{}{
	envoy.ServiceCertType:      nil,
	envoy.RootCertTypeForMTLS:  nil,
	envoy.RootCertTypeForHTTPS: nil,
}

var certTypeToPrefix = map[envoy.SDSCertType]string{
	envoy.ServiceCertType:      serviceCertPrefix,
	envoy.RootCertTypeForMTLS:  rootCertStrictPrefix,
	envoy.RootCertTypeForHTTPS: rootCertRelaxedPrefix,
}

// NewResponse creates a new Secrets Discovery Response.
func NewResponse(_ context.Context, catalog catalog.MeshCataloger, _ smi.MeshSpec, proxy *envoy.Proxy, request *xds.DiscoveryRequest) (*xds.DiscoveryResponse, error) {
	log.Info().Msgf("Composing SDS Discovery Response for proxy: %s", proxy.GetCommonName())

	serviceForProxy, err := catalog.GetServiceFromEnvoyCertificate(proxy.GetCommonName())
	if err != nil {
		log.Error().Err(err).Msgf("Error looking up Service for Envoy with CN=%q", proxy.GetCommonName())
		return nil, err
	}

	cert, err := catalog.GetCertificateForService(*serviceForProxy)
	if err != nil {
		log.Error().Err(err).Msgf("Error obtaining a certificate for client %s", proxy.GetCommonName())
		return nil, err
	}

	// Iterate over the list of tasks and create response structs to be
	// sent to the proxy that made the discovery request
	var resources []*any.Any

	requestedCerts := request.ResourceNames
	log.Trace().Msgf("Received SDS request for ResourceNames (certificates) %+v", requestedCerts)

	// request.ResourceNames is expected to be a list of either "service-cert:namespace/service" or "root-cert:namespace/service"
	for _, envoyProto := range getEnvoySDSSecrets(cert, proxy, requestedCerts, catalog) {
		marshalledSecret, err := ptypes.MarshalAny(envoyProto)
		if err != nil {
			log.Error().Err(err).Msgf("Error marshaling Envoy secret %s for proxy %s for service %s", envoyProto.Name, proxy.GetCommonName(), serviceForProxy.String())
			continue
		}

		resources = append(resources, marshalledSecret)
	}
	return &xds.DiscoveryResponse{
		TypeUrl:   string(envoy.TypeSDS),
		Resources: resources,
	}, nil
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

func getServiceFromRootCertificateRequest(resourceName string, requestedCertType envoy.SDSCertType) (service.NamespacedService, error) {
	// This is a Root Certificate request, which means the resource name begins with "root-cert:"
	// We remove this;  what remains is the namespace and the service separated by a slash:  namespace/service

	slashed := strings.Split(resourceName[len(certTypeToPrefix[requestedCertType]):], "/")
	if len(slashed) != 2 {
		log.Error().Msgf("Error converting %q into a NamespacedService: expected two strings separated by a slash", resourceName)
		return service.NamespacedService{}, errInvalidResourceRequested
	}

	return service.NamespacedService{
		Namespace: slashed[0],
		Service:   slashed[1],
	}, nil
}

func getRequestedCertType(requestedCertificate string) (envoy.SDSCertType, error) {
	// The requestedCertificate is of the format "service-cert:namespace/serviceName"
	// The first string before the colon is the resource certType
	// requestedCertificate could be one of "service-cert" or "root-cert"
	split := strings.Split(requestedCertificate, envoy.Separator)
	if len(split) != 2 {
		log.Error().Msgf("Invalid requestedCertificate requested %q; Expected strings separated by a single colon ':'", requestedCertificate)
		return "", errInvalidResourceName
	}

	certType := envoy.SDSCertType(split[0])
	if _, ok := validResourceTypes[certType]; !ok {
		return "", errInvalidResourceKind
	}

	return certType, nil
}

func getEnvoySDSSecrets(cert certificate.Certificater, proxy *envoy.Proxy, requestedCerts []string, catalog catalog.MeshCataloger) []*auth.Secret {
	// requestedCerts is expected to be a list of either "service-cert:namespace/service" or "root-cert:namespace/service"

	var envoySecrets []*auth.Secret

	svc, err := catalog.GetServiceFromEnvoyCertificate(proxy.GetCommonName())
	if err != nil {
		log.Error().Err(err).Msgf("Error looking up Service for Envoy with CN=%q", proxy.GetCommonName())
		return nil
	}
	serviceForProxy := *svc

	// The Envoy makes a request for a list of resources (aka certificates), which we will send as a response to the SDS request.
	for _, requestedCertificate := range requestedCerts {
		// requestedCertType could be either "service-cert" or "root-cert"
		requestedCertType, err := getRequestedCertType(requestedCertificate)
		if err != nil {
			log.Error().Err(err).Msgf("Invalid resource kind requested: %q", requestedCertificate)
			continue
		}

		if requestedCertType == envoy.ServiceCertType {
			// Make sure that the Envoy requesting a cert for a service indeed belongs to that service
			requestForService, err := getServiceFromServiceCertificateRequest(requestedCertificate)
			if err != nil {
				log.Error().Err(err).Msgf("Error parsing SDS request for certificate type: %q", requestedCertificate)
				continue
			}

			if serviceForProxy != requestForService {
				log.Error().Msgf("Proxy %s (service %s) requested service certificate %s; this is not allowed", proxy.GetCommonName(), serviceForProxy, requestForService)
				continue
			}

			log.Info().Msgf("proxy %s (member of service %s) requested %s", proxy.GetCommonName(), serviceForProxy.String(), requestedCertificate)
			envoySecret, err := getServiceCertSecret(cert, requestedCertificate)
			if err != nil {
				log.Error().Err(err).Msgf("Error creating cert %s for proxy %s for service %s", requestedCertificate, proxy.GetCommonName(), serviceForProxy.String())
				continue
			}
			envoySecrets = append(envoySecrets, envoySecret)
		}

		if requestedCertType == envoy.RootCertTypeForMTLS || requestedCertType == envoy.RootCertTypeForHTTPS {
			// Make sure that the Envoy requesting a cert for a service indeed belongs to that service
			requestForService, err := getServiceFromRootCertificateRequest(requestedCertificate, requestedCertType)
			if err != nil {
				log.Error().Err(err).Msgf("Error parsing SDS request for certificate type: %q", requestedCertificate)
				continue
			}

			if serviceForProxy != requestForService {
				log.Error().Msgf("Proxy %s (service %s) requested service certificate %s; this is not allowed", proxy.GetCommonName(), serviceForProxy, requestForService)
				continue
			}

			log.Info().Msgf("proxy %s (member of service %s) requested %s", proxy.GetCommonName(), serviceForProxy.String(), requestedCertificate)
			envoySecret, err := getRootCert(cert, requestedCertificate, serviceForProxy, catalog, requestedCertType)
			if err != nil {
				log.Error().Err(err).Msgf("Error creating cert %s for proxy %s for service %s", requestedCertificate, proxy.GetCommonName(), serviceForProxy.String())
				continue
			}
			envoySecrets = append(envoySecrets, envoySecret)
		}

	}
	return envoySecrets
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

func getRootCert(cert certificate.Certificater, resourceName string, proxyServiceName service.NamespacedService, mc catalog.MeshCataloger, requestedCertType envoy.SDSCertType) (*auth.Secret, error) {
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
			},
		},
	}

	if requestedCertType == envoy.RootCertTypeForMTLS {
		var matchSANs []*envoy_type_matcher.StringMatcher
		// This block constructs a list of Server Names (peers) that are allowed to connect to the given service.
		// The allowed list is derived from SMI's Traffic Policy.
		serverNames, err := mc.ListAllowedPeerServices(proxyServiceName)
		if err != nil {
			log.Error().Err(err).Msgf("Error getting server names for connected client proxy %s", proxyServiceName)
			return nil, err
		}

		for _, serverName := range serverNames {
			match := envoy_type_matcher.StringMatcher{
				MatchPattern: &envoy_type_matcher.StringMatcher_Exact{
					Exact: serverName.GetCommonName().String(),
				},
			}
			matchSANs = append(matchSANs, &match)
		}
		// Ensure the Subject Alternate Names (SAN) added by CertificateManager.IssueCertificate()
		// matches what is allowed to connect to the downstream service as defined in TrafficPolicy.
		secret.GetValidationContext().MatchSubjectAltNames = matchSANs
	}

	return secret, nil
}
