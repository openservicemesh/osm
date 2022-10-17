package sds

import (
	"fmt"
	"testing"

	xds_auth "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	xds_matcher "github.com/envoyproxy/go-control-plane/envoy/type/matcher/v3"
	"github.com/google/uuid"
	tassert "github.com/stretchr/testify/assert"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/envoy/secrets"
	"github.com/openservicemesh/osm/pkg/models"

	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/service"
)

func TestSecretsBuilder(t *testing.T) {
	assert := tassert.New(t)
	cert := &certificate.Certificate{
		CertChain:  []byte("foo"),
		PrivateKey: []byte("foo"),
		IssuingCA:  []byte("foo"),
		TrustedCAs: []byte("foo"),
	}

	// This is used to dynamically set expectations for each test in the list of table driven tests
	type testCase struct {
		name                        string
		configuredTrustDomains      certificate.TrustDomain
		serviceIdentity             identity.ServiceIdentity
		serviceIdentitiesForService map[service.MeshService][]identity.ServiceIdentity
		expectedSANs                map[string][]string // only set for service-cert
	}

	testCases := []testCase{
		{
			name:                   "test multiple outbound secrets: root-cert-for-mtls-outbound requested",
			configuredTrustDomains: certificate.TrustDomain{Signing: "cluster.local", Validating: "cluster.local"},
			serviceIdentity:        identity.New("sa-1", "ns-1"),
			serviceIdentitiesForService: map[service.MeshService][]identity.ServiceIdentity{
				{
					Name:      "service-2",
					Namespace: "ns-2",
				}: {
					identity.New("sa-2", "ns-2"),
					identity.New("sa-3", "ns-2"),
				},
				{
					Name:      "service-3",
					Namespace: "ns-4",
				}: {
					identity.New("sa-3", "ns-3"),
				},
			},
			expectedSANs: map[string][]string{
				secrets.NameForUpstreamService("service-2", "ns-2"): {"sa-2.ns-2.cluster.local", "sa-3.ns-2.cluster.local"},
				secrets.NameForUpstreamService("service-3", "ns-4"): {"sa-3.ns-3.cluster.local"},
			},
		},
		{
			name:                   "test no outbound secrets",
			configuredTrustDomains: certificate.TrustDomain{Signing: "cluster.local", Validating: "cluster.local"},
			serviceIdentity:        identity.New("sa-1", "ns-1"),
		},
		{
			name:                   "test multiple outbound secrets with multiple trust domains",
			serviceIdentity:        identity.New("sa-1", "ns-1"),
			configuredTrustDomains: certificate.TrustDomain{Signing: "cluster.local", Validating: "cluster.new"},
			serviceIdentitiesForService: map[service.MeshService][]identity.ServiceIdentity{
				{
					Name:      "service-2",
					Namespace: "ns-2",
				}: {
					identity.New("sa-2", "ns-2"),
					identity.New("sa-3", "ns-2"),
				},
				{
					Name:      "service-3",
					Namespace: "ns-4",
				}: {
					identity.New("sa-3", "ns-3"),
				},
			},
			expectedSANs: map[string][]string{
				secrets.NameForUpstreamService("service-2", "ns-2"): {"sa-2.ns-2.cluster.local", "sa-3.ns-2.cluster.local", "sa-2.ns-2.cluster.new", "sa-3.ns-2.cluster.new"},
				secrets.NameForUpstreamService("service-3", "ns-4"): {"sa-3.ns-3.cluster.local", "sa-3.ns-3.cluster.new"},
			},
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("Testing test case %d: %s", i, tc.name), func(t *testing.T) {
			builder := NewBuilder()
			proxy := models.NewProxy(models.KindSidecar, uuid.New(), identity.New("sa-1", "ns-1"), nil, 1)
			builder.SetProxy(proxy).SetProxyCert(cert).SetTrustDomain(tc.configuredTrustDomains)

			builder.SetServiceIdentitiesForService(tc.serviceIdentitiesForService)

			sdsSecrets := builder.Build()
			assert.Len(sdsSecrets, 2+len(tc.serviceIdentitiesForService))

			serviceSecret := sdsSecrets[0]
			assert.NotNil(serviceSecret.GetTlsCertificate().GetCertificateChain().GetInlineBytes())
			assert.NotNil(serviceSecret.GetTlsCertificate().GetPrivateKey().GetInlineBytes())

			inboundValidationSecret := sdsSecrets[1]
			assert.NotNil(inboundValidationSecret.GetValidationContext().GetTrustedCa().GetInlineBytes())

			for _, outboundSecret := range sdsSecrets[2:] {
				// outbound cert:
				actualSANs := subjectAltNamesToStr(outboundSecret.GetValidationContext().GetMatchTypedSubjectAltNames())
				assert.NotNil(outboundSecret.GetValidationContext().GetTrustedCa().GetInlineBytes())
				assert.ElementsMatch(actualSANs, tc.expectedSANs[outboundSecret.GetName()])
			}
		})
	}
}

func TestGetSubjectAltNamesFromSvcAccount(t *testing.T) {
	type testCase struct {
		serviceIdentities   []identity.ServiceIdentity
		expectedSANMatchers []*xds_auth.SubjectAltNameMatcher
	}

	testCases := []testCase{
		{
			serviceIdentities: []identity.ServiceIdentity{
				identity.K8sServiceAccount{Name: "sa-1", Namespace: "ns-1"}.ToServiceIdentity(),
				identity.K8sServiceAccount{Name: "sa-2", Namespace: "ns-2"}.ToServiceIdentity(),
			},
			expectedSANMatchers: []*xds_auth.SubjectAltNameMatcher{
				{
					SanType: xds_auth.SubjectAltNameMatcher_DNS,
					Matcher: &xds_matcher.StringMatcher{
						MatchPattern: &xds_matcher.StringMatcher_Exact{
							Exact: "sa-1.ns-1.cluster.local",
						},
					},
				},
				{
					SanType: xds_auth.SubjectAltNameMatcher_DNS,
					Matcher: &xds_matcher.StringMatcher{
						MatchPattern: &xds_matcher.StringMatcher_Exact{
							Exact: "sa-2.ns-2.cluster.local",
						},
					},
				},
			},
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("Testing test case %d", i), func(t *testing.T) {
			assert := tassert.New(t)

			actual := getSubjectAltNamesFromSvcIdentities(tc.serviceIdentities, certificate.TrustDomain{Signing: "cluster.local", Validating: "cluster.local"})
			assert.ElementsMatch(actual, tc.expectedSANMatchers)
		})
	}
}

func TestSubjectAltNamesToStr(t *testing.T) {
	type testCase struct {
		sanMatchers []*xds_auth.SubjectAltNameMatcher
		strSANs     []string
	}

	testCases := []testCase{
		{
			sanMatchers: []*xds_auth.SubjectAltNameMatcher{
				{
					SanType: xds_auth.SubjectAltNameMatcher_DNS,
					Matcher: &xds_matcher.StringMatcher{
						MatchPattern: &xds_matcher.StringMatcher_Exact{
							Exact: "sa-1.ns-1.cluster.local",
						},
					},
				},
				{
					SanType: xds_auth.SubjectAltNameMatcher_DNS,
					Matcher: &xds_matcher.StringMatcher{
						MatchPattern: &xds_matcher.StringMatcher_Exact{
							Exact: "sa-2.ns-2.cluster.local",
						},
					},
				},
			},
			strSANs: []string{
				"sa-1.ns-1.cluster.local",
				"sa-2.ns-2.cluster.local",
			},
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("Testing test case %d", i), func(t *testing.T) {
			assert := tassert.New(t)

			actual := subjectAltNamesToStr(tc.sanMatchers)
			assert.ElementsMatch(actual, tc.strSANs)
		})
	}
}

func subjectAltNamesToStr(sanMatchList []*xds_auth.SubjectAltNameMatcher) []string {
	var sanStr []string

	for _, sanMatcher := range sanMatchList {
		sanStr = append(sanStr, sanMatcher.Matcher.GetExact())
	}
	return sanStr
}
