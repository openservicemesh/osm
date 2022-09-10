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

	"github.com/openservicemesh/osm/pkg/envoy"
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
		serviceIdentity             identity.ServiceIdentity
		serviceIdentitiesForService map[service.MeshService][]identity.ServiceIdentity
		// expectations
		expectedSANs map[string][]string // only set for service-cert
	}

	testCases := []testCase{
		// Test case 2: root-cert-for-mtls-outbound requested -------------------------------
		{
			name:            "test multiple outbound secrets",
			serviceIdentity: identity.New("sa-1", "ns-1"),
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
			// expectations
			expectedSANs: map[string][]string{
				secrets.NameForUpstreamService("service-2", "ns-2"): {"sa-2.ns-2.cluster.local", "sa-3.ns-2.cluster.local"},
				secrets.NameForUpstreamService("service-3", "ns-4"): {"sa-3.ns-3.cluster.local"},
			},
		},
		// Test case 2 end -------------------------------

		// Test case 3: service-cert requested -------------------------------
		{
			name:            "test no outbound secrets",
			serviceIdentity: identity.New("sa-1", "ns-1"),
		},
		// Test case 3 end -------------------------------
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("Testing test case %d: %s", i, tc.name), func(t *testing.T) {
			builder := NewBuilder()
			proxy := envoy.NewProxy(envoy.KindSidecar, uuid.New(), identity.New("sa-1", "ns-1"), nil, 1)
			builder.SetProxy(proxy).SetProxyCert(cert).SetTrustDomain("cluster.local")

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

			actual := getSubjectAltNamesFromSvcIdentities(tc.serviceIdentities, "cluster.local")
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
