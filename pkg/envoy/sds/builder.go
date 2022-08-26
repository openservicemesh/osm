package sds

import (
	xds_core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	xds_auth "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	xds_matcher "github.com/envoyproxy/go-control-plane/envoy/type/matcher/v3"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/envoy/secrets"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/service"
)

type sdsBuilder struct {
	proxy *envoy.Proxy

	// Service certificate for this proxy
	serviceCert *certificate.Certificate

	trustDomain string

	// identities, used for SAN matches, mapped to the name of the secret. Currently only used for outbound secrets.
	identitiesForSecrets map[string][]identity.ServiceIdentity
}

// NewBuilder returns a new sdsBuilder
func NewBuilder() *sdsBuilder { //nolint: revive // unexported-return
	return &sdsBuilder{}
}

func (b *sdsBuilder) SetProxy(proxy *envoy.Proxy) *sdsBuilder {
	b.proxy = proxy
	return b
}

func (b *sdsBuilder) SetProxyCert(cert *certificate.Certificate) *sdsBuilder {
	b.serviceCert = cert
	return b
}

func (b *sdsBuilder) SetTrustDomain(trustDomain string) *sdsBuilder {
	b.trustDomain = trustDomain
	return b
}

func (b *sdsBuilder) SetServiceIdentitiesForService(serviceIdentitiesForServices map[service.MeshService][]identity.ServiceIdentity) *sdsBuilder {
	b.identitiesForSecrets = make(map[string][]identity.ServiceIdentity)
	for svc, serviceIdentities := range serviceIdentitiesForServices {
		b.identitiesForSecrets[secrets.NameForUpstreamService(svc.Name, svc.Namespace)] = serviceIdentities
	}
	return b
}

// Build generates SDS Secret Resources based on requested certs in the DiscoveryRequest
func (b *sdsBuilder) Build() []*xds_auth.Secret {
	var sdsResources = make([]*xds_auth.Secret, 0, len(b.identitiesForSecrets))

	sdsResources = append(sdsResources, b.buildServiceSecret())
	// SAN validation should not be performed by the root validation certificate used by the upstream server
	// to validate a downstream client. This is because of the following:
	// 1. SAN validation is already performed by the RBAC filter on the inbound listener's filter chain (using
	//    network RBAC filter) and each HTTP route in the inbound route ocnfiguration (using HTTP RBAC per route).
	// 2. The same root validation certificate is used to validate both in-mesh and ingress downstreams.
	// For these reasons, we only perform SAN validation of peer certificates on downstream clients (ie. outbound SAN
	// validation).
	sdsResources = append(sdsResources, b.buildSecret(secrets.NameForMTLSInbound, nil))

	for name, identites := range b.identitiesForSecrets {
		sdsResources = append(sdsResources, b.buildSecret(name, identites))
	}
	return sdsResources
}

// buildServiceCertSecret creates the struct with certificates for the service, which the
// connected Envoy proxy belongs to.
func (b *sdsBuilder) buildServiceSecret() *xds_auth.Secret {
	return &xds_auth.Secret{
		// The Name field must match the tls_context.common_tls_context.tls_certificate_sds_secret_configs.name in the Envoy yaml config
		Name: secrets.NameForIdentity(b.proxy.Identity),
		Type: &xds_auth.Secret_TlsCertificate{
			TlsCertificate: &xds_auth.TlsCertificate{
				CertificateChain: &xds_core.DataSource{
					Specifier: &xds_core.DataSource_InlineBytes{
						InlineBytes: b.serviceCert.GetCertificateChain(),
					},
				},
				PrivateKey: &xds_core.DataSource{
					Specifier: &xds_core.DataSource_InlineBytes{
						InlineBytes: b.serviceCert.GetPrivateKey(),
					},
				},
			},
		},
	}
}

func (b *sdsBuilder) buildSecret(name string, allowedIdentities []identity.ServiceIdentity) *xds_auth.Secret {
	secret := &xds_auth.Secret{
		// The Name field must match the tls_context.common_tls_context.tls_certificate_sds_secret_configs.name
		Name: name,
		Type: &xds_auth.Secret_ValidationContext{
			ValidationContext: &xds_auth.CertificateValidationContext{
				TrustedCa: &xds_core.DataSource{
					Specifier: &xds_core.DataSource_InlineBytes{
						InlineBytes: b.serviceCert.GetTrustedCAs(),
					},
				},
			},
		},
	}
	secret.GetValidationContext().MatchTypedSubjectAltNames = getSubjectAltNamesFromSvcIdentities(allowedIdentities, b.trustDomain)
	return secret
}

// Note: ServiceIdentity must be in the format "name.namespace" [https://github.com/openservicemesh/osm/issues/3188]
func getSubjectAltNamesFromSvcIdentities(serviceIdentities []identity.ServiceIdentity, trustDomain string) []*xds_auth.SubjectAltNameMatcher {
	var matchSANs []*xds_auth.SubjectAltNameMatcher

	for _, si := range serviceIdentities {
		match := xds_auth.SubjectAltNameMatcher{
			SanType: xds_auth.SubjectAltNameMatcher_DNS,
			Matcher: &xds_matcher.StringMatcher{
				MatchPattern: &xds_matcher.StringMatcher_Exact{
					Exact: si.AsPrincipal(trustDomain),
				},
			},
		}
		matchSANs = append(matchSANs, &match)
	}

	return matchSANs
}
