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
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/service"
)

// NewResponse creates a new Secrets Discovery Response.
func NewResponse(meshCatalog catalog.MeshCataloger, proxy *envoy.Proxy, request *xds_discovery.DiscoveryRequest, cfg configurator.Configurator, certManager certificate.Manager) (*xds_discovery.DiscoveryResponse, error) {
	log.Info().Msgf("Composing SDS Discovery Response for proxy: %s", proxy.GetCommonName())

	svcList, err := meshCatalog.GetServicesFromEnvoyCertificate(proxy.GetCommonName())
	if err != nil {
		log.Error().Err(err).Msgf("Error getting services associated with proxy %s", proxy.GetCommonName())
		return nil, err
	}

	// OSM currently relies on kubernetes ServiceAccount for service identity
	svcAccount, err := catalog.GetServiceAccountFromProxyCertificate(proxy.GetCommonName())
	if err != nil {
		log.Error().Err(err).Msgf("Error retrieving SerivceAccount for proxy %s", proxy.GetCommonName())
		return nil, err
	}

	sdsImpl := newSDSImpl(proxy, meshCatalog, certManager, cfg, svcList, svcAccount)
	return sdsImpl.createDiscoveryResponse(request)
}

func newSDSImpl(proxy *envoy.Proxy, meshCatalog catalog.MeshCataloger, certManager certificate.Manager, cfg configurator.Configurator, proxyServices []service.MeshService, svcAccount service.K8sServiceAccount) *sdsImpl {
	impl := &sdsImpl{
		proxy:         proxy,
		meshCatalog:   meshCatalog,
		certManager:   certManager,
		cfg:           cfg,
		svcAccount:    svcAccount,
		proxyServices: proxyServices,
	}

	return impl
}

func (s *sdsImpl) createDiscoveryResponse(request *xds_discovery.DiscoveryRequest) (*xds_discovery.DiscoveryResponse, error) {
	// Resources corresponding to SDS secrets returned as a part of the DiscoveryResponse
	var resources []*any.Any

	// The DisvoceryRequest contains the requested certs
	requestedCerts := request.ResourceNames
	log.Trace().Msgf("Received SDS request for ResourceNames (certificates) %+v on proxy %s, services: %v", requestedCerts, s.proxy.GetCommonName(), s.proxyServices)

	for _, proxyService := range s.proxyServices {
		log.Trace().Msgf("Creating SDS config for proxy service %s on proxy %s", proxyService, s.proxy.GetCommonName())
		// 1. Issue a service certificate for this proxy
		// OSM currently relies on kubernetes ServiceAccount for service identity
		si := identity.GetKubernetesServiceIdentity(s.svcAccount, identity.ClusterLocalTrustDomain)
		cert, err := s.certManager.IssueCertificate(certificate.CommonName(si), s.cfg.GetServiceCertValidityPeriod())
		if err != nil {
			log.Error().Err(err).Msgf("Error issuing a certificate for proxy service %s", proxyService)
			return nil, err
		}

		// 2. Create SDS secret resources based on the requested certs in the DiscoveryRequest
		// request.ResourceNames is expected to be a list of either "service-cert:namespace/service" or "root-cert:namespace/service"
		for _, envoyProto := range s.getSDSSecrets(cert, requestedCerts, proxyService) {
			marshalledSecret, err := ptypes.MarshalAny(envoyProto)
			if err != nil {
				log.Error().Err(err).Msgf("Error marshaling Envoy secret %s for proxy %s for service %s", envoyProto.Name, s.proxy.GetCommonName(), proxyService)
				continue
			}

			resources = append(resources, marshalledSecret)
		}
	}

	return &xds_discovery.DiscoveryResponse{
		TypeUrl:   string(envoy.TypeSDS),
		Resources: resources,
	}, nil
}

func (s *sdsImpl) getSDSSecrets(cert certificate.Certificater, requestedCerts []string, proxyService service.MeshService) []*xds_auth.Secret {
	// requestedCerts is expected to be a list of either of the following:
	// - "service-cert:namespace/service"
	// - "root-cert-for-mtls-outbound:namespace/service"
	// - "root-cert-for-mtls-inbound:namespace/service"
	// - "root-cert-for-https:namespace/service"

	var envoySecrets []*xds_auth.Secret

	// The Envoy makes a request for a list of resources (aka certificates), which we will send as a response to the SDS request.
	for _, requestedCertificate := range requestedCerts {
		// requestedCertType could be either "service-cert" or "root-cert"
		sdsCert, err := envoy.UnmarshalSDSCert(requestedCertificate)
		if err != nil {
			log.Error().Err(err).Msgf("Invalid resource kind requested: %q", requestedCertificate)
			continue
		}

		log.Debug().Msgf("proxy %s (member of service %s) requested %s", s.proxy.GetCommonName(), proxyService, requestedCertificate)

		switch sdsCert.CertType {
		case envoy.ServiceCertType:
			// A service certificate is requested
			envoySecret, err := getServiceCertSecret(cert, requestedCertificate)
			if err != nil {
				log.Error().Err(err).Msgf("Error creating cert %s for proxy %s for service %s", requestedCertificate, s.proxy.GetCommonName(), proxyService)
				continue
			}
			envoySecrets = append(envoySecrets, envoySecret)

		// A root certificate used to validate a service certificate is requested
		case envoy.RootCertTypeForMTLSInbound, envoy.RootCertTypeForMTLSOutbound, envoy.RootCertTypeForHTTPS:
			envoySecret, err := s.getRootCert(cert, *sdsCert, proxyService)
			if err != nil {
				log.Error().Err(err).Msgf("Error creating cert %s for proxy %s for service %s", requestedCertificate, s.proxy.GetCommonName(), proxyService)
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

func (s *sdsImpl) getRootCert(cert certificate.Certificater, sdscert envoy.SDSCert, proxyService service.MeshService) (*xds_auth.Secret, error) {
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

	if s.cfg.IsPermissiveTrafficPolicyMode() {
		// In permissive mode, there are no SMI TrafficTarget policies, so
		// SAN matching is not required.
		return secret, nil
	}

	// Program SAN matching based on SMI TrafficTarget policies
	switch sdscert.CertType {
	case envoy.RootCertTypeForMTLSOutbound:
		// For outbound certificate validation context, the SAN needs to the list of service identities
		// corresponding to the upstream service. This means, if the sdscert.MeshService points to 'X',
		// the SANs for this certificate should correspond to the service identities of 'X'.
		svcAccounts, err := s.meshCatalog.ListServiceAccountsForService(sdscert.MeshService)
		if err != nil {
			log.Error().Err(err).Msgf("Error listing service accounts for service %q", sdscert.MeshService)
			return nil, err
		}
		secret.GetValidationContext().MatchSubjectAltNames = getSubjectAltNamesFromSvcAccount(svcAccounts)
		log.Trace().Msgf("Proxy for service=%s, upstream cert=%s will only allow SANs exactly matching: %v",
			proxyService, sdscert, subjectAltNamesToStr(secret.GetValidationContext().GetMatchSubjectAltNames()))

	case envoy.RootCertTypeForMTLSInbound:
		// For inbound certificate validation context, the SAN needs to be the list of all downstream
		// service identities that are allowed to connect to this upstream service. This means, if sdscert.MeshService
		// points to 'X', the SANs for this certificate should correspond to all the downstream service identities
		// allowed to reach 'X'.
		svcAccounts, err := s.meshCatalog.ListAllowedInboundServiceAccounts(s.svcAccount)
		if err != nil {
			log.Error().Err(err).Msgf("Error listing inbound service accounts for proxy service %s", proxyService)
			return nil, err
		}
		secret.GetValidationContext().MatchSubjectAltNames = getSubjectAltNamesFromSvcAccount(svcAccounts)
		log.Trace().Msgf("Proxy for service=%s, downstream cert=%s will only allow SANs exactly matching: %v",
			proxyService, sdscert, subjectAltNamesToStr(secret.GetValidationContext().GetMatchSubjectAltNames()))

	default:
		log.Debug().Msgf("SAN matching not needed for cert %s", sdscert)
	}

	return secret, nil
}

func getSubjectAltNamesFromSvcAccount(svcAccounts []service.K8sServiceAccount) []*xds_matcher.StringMatcher {
	var matchSANs []*xds_matcher.StringMatcher

	for _, svcAccount := range svcAccounts {
		// OSM currently relies on kubernetes ServiceAccount for service identity
		si := identity.GetKubernetesServiceIdentity(svcAccount, identity.ClusterLocalTrustDomain)
		match := xds_matcher.StringMatcher{
			MatchPattern: &xds_matcher.StringMatcher_Exact{
				Exact: si.String(),
			},
		}
		matchSANs = append(matchSANs, &match)
	}

	return matchSANs
}

func subjectAltNamesToStr(sanMatchList []*xds_matcher.StringMatcher) []string {
	var sanStr []string

	for _, sanMatcher := range sanMatchList {
		sanStr = append(sanStr, sanMatcher.GetExact())
	}
	return sanStr
}
