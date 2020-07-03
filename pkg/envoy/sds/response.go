package sds

import (
	"context"

	xds "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	auth "github.com/envoyproxy/go-control-plane/envoy/api/v2/auth"
	core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	envoy_type_matcher "github.com/envoyproxy/go-control-plane/envoy/type/matcher"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"

	"github.com/open-service-mesh/osm/pkg/catalog"
	"github.com/open-service-mesh/osm/pkg/certificate"
	"github.com/open-service-mesh/osm/pkg/configurator"
	"github.com/open-service-mesh/osm/pkg/envoy"
	"github.com/open-service-mesh/osm/pkg/service"
	"github.com/open-service-mesh/osm/pkg/smi"
)

// Used mainly for logging/translation purposes
var directionMap = map[envoy.SDSCertType]string{
	envoy.RootCertTypeForMTLSInbound:  "inbound",
	envoy.RootCertTypeForMTLSOutbound: "outbound",
}

// NewResponse creates a new Secrets Discovery Response.
func NewResponse(_ context.Context, catalog catalog.MeshCataloger, _ smi.MeshSpec, proxy *envoy.Proxy, request *xds.DiscoveryRequest, config *configurator.Config) (*xds.DiscoveryResponse, error) {
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
		sdsCert, err := envoy.UnmarshalSDSCert(requestedCertificate)
		if err != nil {
			log.Error().Err(err).Msgf("Invalid resource kind requested: %q", requestedCertificate)
			continue
		}

		if serviceForProxy != sdsCert.Service {
			log.Error().Msgf("Proxy %s (service %s) requested service certificate %s; this is not allowed", proxy.GetCommonName(), serviceForProxy, sdsCert.Service)
			continue
		}

		switch sdsCert.CertType {
		case envoy.ServiceCertType:
			{
				log.Info().Msgf("proxy %s (member of service %s) requested %s", proxy.GetCommonName(), serviceForProxy.String(), requestedCertificate)
				envoySecret, err := getServiceCertSecret(cert, requestedCertificate)
				if err != nil {
					log.Error().Err(err).Msgf("Error creating cert %s for proxy %s for service %s", requestedCertificate, proxy.GetCommonName(), serviceForProxy.String())
					continue
				}
				envoySecrets = append(envoySecrets, envoySecret)
			}
		case envoy.RootCertTypeForMTLSInbound:
			fallthrough
		case envoy.RootCertTypeForMTLSOutbound:
			fallthrough
		case envoy.RootCertTypeForHTTPS:
			{
				log.Info().Msgf("proxy %s (member of service %s) requested %s", proxy.GetCommonName(), serviceForProxy.String(), requestedCertificate)
				envoySecret, err := getRootCert(cert, *sdsCert, serviceForProxy, catalog)
				if err != nil {
					log.Error().Err(err).Msgf("Error creating cert %s for proxy %s for service %s", requestedCertificate, proxy.GetCommonName(), serviceForProxy.String())
					continue
				}
				envoySecrets = append(envoySecrets, envoySecret)
			}
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

func getRootCert(cert certificate.Certificater, sdscert envoy.SDSCert, proxyServiceName service.NamespacedService, mc catalog.MeshCataloger) (*auth.Secret, error) {
	secret := &auth.Secret{
		// The Name field must match the tls_context.common_tls_context.tls_certificate_sds_secret_configs.name
		Name: sdscert.String(),
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

	switch sdscert.CertType {
	case envoy.RootCertTypeForMTLSOutbound:
		fallthrough
	case envoy.RootCertTypeForMTLSInbound:
		{
			var matchSANs []*envoy_type_matcher.StringMatcher
			var serverNames []service.NamespacedService
			var err error

			// This block constructs a list of Server Names (peers) that are allowed to connect to the given service.
			// The allowed list is derived from SMI's Traffic Policy.
			if sdscert.CertType == envoy.RootCertTypeForMTLSOutbound {
				// Outbound
				serverNames, err = mc.ListAllowedOutboundServices(proxyServiceName)
			} else {
				// Inbound
				serverNames, err = mc.ListAllowedInboundServices(proxyServiceName)
			}

			if err != nil {
				log.Error().Err(err).Msgf("Error getting server names for %s allowed Services %s", directionMap[sdscert.CertType], proxyServiceName)
				return nil, err
			}

			var matchingCerts []string
			for _, serverName := range serverNames {
				matchingCerts = append(matchingCerts, serverName.GetCommonName().String())
				match := envoy_type_matcher.StringMatcher{
					MatchPattern: &envoy_type_matcher.StringMatcher_Exact{
						Exact: serverName.GetCommonName().String(),
					},
				}
				matchSANs = append(matchSANs, &match)
			}

			log.Trace().Msgf("Proxy for service %s will only allow %s SANs exactly matching: %+v", proxyServiceName, directionMap[sdscert.CertType], matchingCerts)

			// Ensure the Subject Alternate Names (SAN) added by CertificateManager.IssueCertificate()
			// matches what is allowed to connect to the downstream service as defined in TrafficPolicy.
			secret.GetValidationContext().MatchSubjectAltNames = matchSANs
		}
	default:
		// Nothing here
	}

	return secret, nil
}
