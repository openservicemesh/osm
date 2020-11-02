package sds

import (
	xds_core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	xds_auth "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	xds_discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	xds_matcher "github.com/envoyproxy/go-control-plane/envoy/type/matcher/v3"

	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"

	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/service"
)

// Used mainly for logging/translation purposes
var directionMap = map[envoy.SDSCertType]string{
	envoy.RootCertTypeForMTLSInbound:  "inbound",
	envoy.RootCertTypeForMTLSOutbound: "outbound",
}

// NewResponse creates a new Secrets Discovery Response.
func NewResponse(catalog catalog.MeshCataloger, proxy *envoy.Proxy, request *xds_discovery.DiscoveryRequest, cfg configurator.Configurator, certManager certificate.Manager) (*xds_discovery.DiscoveryResponse, error) {
	log.Info().Msgf("Composing SDS Discovery Response for proxy: %s", proxy.GetCommonName())

	svcList, err := catalog.GetServicesFromEnvoyCertificate(proxy.GetCommonName())
	// Github Issue #1575
	serviceForProxy := svcList[0]
	if err != nil {
		log.Error().Err(err).Msgf("Error looking up service for Envoy with CN=%q", proxy.GetCommonName())
		return nil, err
	}

	cert, err := certManager.IssueCertificate(serviceForProxy.GetCommonName(), cfg.GetServiceCertValidityPeriod())
	if err != nil {
		log.Error().Err(err).Msgf("Error issuing a certificate for proxy service %s", serviceForProxy)
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
	return &xds_discovery.DiscoveryResponse{
		TypeUrl:   string(envoy.TypeSDS),
		Resources: resources,
	}, nil
}

func getEnvoySDSSecrets(cert certificate.Certificater, proxy *envoy.Proxy, requestedCerts []string, catalog catalog.MeshCataloger) []*xds_auth.Secret {
	// requestedCerts is expected to be a list of either "service-cert:namespace/service" or "root-cert:namespace/service"

	var envoySecrets []*xds_auth.Secret

	svcList, err := catalog.GetServicesFromEnvoyCertificate(proxy.GetCommonName())
	if err != nil {
		log.Error().Err(err).Msgf("Error looking up service for Envoy with CN=%q", proxy.GetCommonName())
		return nil
	}
	// Github Issue #1575
	serviceForProxy := svcList[0]

	// The Envoy makes a request for a list of resources (aka certificates), which we will send as a response to the SDS request.
	for _, requestedCertificate := range requestedCerts {
		// requestedCertType could be either "service-cert" or "root-cert"
		sdsCert, err := envoy.UnmarshalSDSCert(requestedCertificate)
		if err != nil {
			log.Error().Err(err).Msgf("Invalid resource kind requested: %q", requestedCertificate)
			continue
		}

		if serviceForProxy != sdsCert.MeshService {
			log.Error().Msgf("Proxy %s (service %s) requested service certificate %s; this is not allowed", proxy.GetCommonName(), serviceForProxy, sdsCert.MeshService)
			continue
		}

		switch sdsCert.CertType {
		case envoy.ServiceCertType:
			log.Info().Msgf("proxy %s (member of service %s) requested %s", proxy.GetCommonName(), serviceForProxy.String(), requestedCertificate)
			envoySecret, err := getServiceCertSecret(cert, requestedCertificate)
			if err != nil {
				log.Error().Err(err).Msgf("Error creating cert %s for proxy %s for service %s", requestedCertificate, proxy.GetCommonName(), serviceForProxy.String())
				continue
			}
			envoySecrets = append(envoySecrets, envoySecret)

		case envoy.RootCertTypeForMTLSInbound:
			fallthrough
		case envoy.RootCertTypeForMTLSOutbound:
			fallthrough
		case envoy.RootCertTypeForHTTPS:
			log.Info().Msgf("proxy %s (member of service %s) requested %s", proxy.GetCommonName(), serviceForProxy.String(), requestedCertificate)
			envoySecret, err := getRootCert(cert, *sdsCert, serviceForProxy, catalog)
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
func getServiceCertSecret(cert certificate.Certificater, name string) (*xds_auth.Secret, error) {
	secret := &xds_auth.Secret{
		// The Name field must match the tls_context.common_tls_context.tls_certificate_sds_secret_configs.name in the Envoy yaml config
		Name: name,
		Type: &xds_auth.Secret_TlsCertificate{
			TlsCertificate: &xds_auth.TlsCertificate{
				CertificateChain: &xds_core.DataSource{
					Specifier: &xds_core.DataSource_InlineBytes{
						InlineBytes: cert.GetCertificateChain(),
					},
				},
				PrivateKey: &xds_core.DataSource{
					Specifier: &xds_core.DataSource_InlineBytes{
						InlineBytes: cert.GetPrivateKey(),
					},
				},
			},
		},
	}
	return secret, nil
}

func getRootCert(cert certificate.Certificater, sdscert envoy.SDSCert, proxyServiceName service.MeshService, mc catalog.MeshCataloger) (*xds_auth.Secret, error) {
	secret := &xds_auth.Secret{
		// The Name field must match the tls_context.common_tls_context.tls_certificate_sds_secret_configs.name
		Name: sdscert.String(),
		Type: &xds_auth.Secret_ValidationContext{
			ValidationContext: &xds_auth.CertificateValidationContext{
				TrustedCa: &xds_core.DataSource{
					Specifier: &xds_core.DataSource_InlineBytes{
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
		var matchSANs []*xds_matcher.StringMatcher
		var serverNames []service.MeshService
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
			match := xds_matcher.StringMatcher{
				MatchPattern: &xds_matcher.StringMatcher_Exact{
					Exact: serverName.GetCommonName().String(),
				},
			}
			matchSANs = append(matchSANs, &match)
		}

		log.Trace().Msgf("Proxy for service %s will only allow %s SANs exactly matching: %+v", proxyServiceName, directionMap[sdscert.CertType], matchingCerts)

		// Ensure the Subject Alternate Names (SAN) added by CertificateManager.IssueCertificate()
		// matches what is allowed to connect to the downstream service as defined in TrafficPolicy.
		secret.GetValidationContext().MatchSubjectAltNames = matchSANs
	default:
		log.Debug().Msgf("SAN matching not needed for cert type %s", sdscert.CertType.String())
	}

	return secret, nil
}
