package sds

import (
	"fmt"
	"testing"
	"time"

	xds_auth "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	xds_discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	xds_matcher "github.com/envoyproxy/go-control-plane/envoy/type/matcher/v3"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	tassert "github.com/stretchr/testify/assert"

	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/envoy/secrets"

	"github.com/openservicemesh/osm/pkg/certificate"
	tresorFake "github.com/openservicemesh/osm/pkg/certificate/providers/tresor/fake"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/service"
)

// TestNewResponse sets up a fake kube client, then a pod and makes an SDS request,
// and finally verifies the response from sds.NewResponse().
func TestNewResponse(t *testing.T) {
	assert := tassert.New(t)
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	certManager := tresorFake.NewFake(1 * time.Hour)

	// We deliberately set the namespace and service accounts to random values
	// to ensure no hard-coded values sneak in.
	proxySvcID := identity.New(uuid.New().String(), uuid.New().String())

	// This is the thing we are going to be requesting (pretending that the Envoy is requesting it)
	testCases := []struct {
		name                        string
		serviceIdentitiesForService map[service.MeshService][]identity.ServiceIdentity
		requestedCerts              []string
		trustDomain                 string
		expectedCertToSAN           map[string][]string
	}{
		{
			name: "requested certs match all certs",
			serviceIdentitiesForService: map[service.MeshService][]identity.ServiceIdentity{
				{Name: "svc-1", Namespace: "ns-1"}: {
					identity.New("sa-1", "ns-1"),
					identity.New("sa-2", "ns-1"),
				},
				{Name: "svc-A", Namespace: "ns-A"}: {
					identity.New("sa-A", "ns-A"),
				},
			},
			requestedCerts: []string{
				secrets.NameForUpstreamService("svc-1", "ns-1"),
				secrets.NameForUpstreamService("svc-A", "ns-A"),
				secrets.NameForIdentity(proxySvcID),
				secrets.NameForMTLSInbound,
			},
			expectedCertToSAN: map[string][]string{
				secrets.NameForUpstreamService("svc-1", "ns-1"): {
					"sa-1.ns-1.cluster.local",
					"sa-2.ns-1.cluster.local",
				},
				secrets.NameForUpstreamService("svc-A", "ns-A"): {
					"sa-A.ns-A.cluster.local",
				},
				secrets.NameForIdentity(proxySvcID): nil,
				secrets.NameForMTLSInbound:          nil,
			},
		},
		{
			name: "requested certs missing inbound cert",
			serviceIdentitiesForService: map[service.MeshService][]identity.ServiceIdentity{
				{Name: "svc-1", Namespace: "ns-1"}: {
					identity.New("sa-1", "ns-1"),
					identity.New("sa-2", "ns-1"),
				},
				{Name: "svc-A", Namespace: "ns-A"}: {
					identity.New("sa-A", "ns-A"),
				},
			},
			requestedCerts: []string{
				secrets.NameForUpstreamService("svc-1", "ns-1"),
				secrets.NameForUpstreamService("svc-A", "ns-A"),
				secrets.NameForIdentity(proxySvcID),
			},
			expectedCertToSAN: map[string][]string{
				secrets.NameForUpstreamService("svc-1", "ns-1"): {
					"sa-1.ns-1.cluster.local",
					"sa-2.ns-1.cluster.local",
				},
				secrets.NameForUpstreamService("svc-A", "ns-A"): {
					"sa-A.ns-A.cluster.local",
				},
				secrets.NameForIdentity(proxySvcID): nil,
			},
		},
		{
			name: "requested certs missing 1 outbound cert",
			serviceIdentitiesForService: map[service.MeshService][]identity.ServiceIdentity{
				{Name: "svc-1", Namespace: "ns-1"}: {
					identity.New("sa-1", "ns-1"),
					identity.New("sa-2", "ns-1"),
				},
				{Name: "svc-A", Namespace: "ns-A"}: {
					identity.New("sa-A", "ns-A"),
				},
			},
			requestedCerts: []string{
				secrets.NameForUpstreamService("svc-A", "ns-A"),
				secrets.NameForIdentity(proxySvcID),
				secrets.NameForMTLSInbound,
			},
			expectedCertToSAN: map[string][]string{
				secrets.NameForUpstreamService("svc-A", "ns-A"): {
					"sa-A.ns-A.cluster.local",
				},
				secrets.NameForIdentity(proxySvcID): nil,
				secrets.NameForMTLSInbound:          nil,
			},
		},
		{
			name: "requested certs missing service cert",
			serviceIdentitiesForService: map[service.MeshService][]identity.ServiceIdentity{
				{Name: "svc-1", Namespace: "ns-1"}: {
					identity.New("sa-1", "ns-1"),
					identity.New("sa-2", "ns-1"),
				},
				{Name: "svc-A", Namespace: "ns-A"}: {
					identity.New("sa-A", "ns-A"),
				},
			},
			requestedCerts: []string{
				secrets.NameForUpstreamService("svc-1", "ns-1"),
				secrets.NameForUpstreamService("svc-A", "ns-A"),
				secrets.NameForMTLSInbound,
			},
			expectedCertToSAN: map[string][]string{
				secrets.NameForUpstreamService("svc-1", "ns-1"): {
					"sa-1.ns-1.cluster.local",
					"sa-2.ns-1.cluster.local",
				},
				secrets.NameForUpstreamService("svc-A", "ns-A"): {
					"sa-A.ns-A.cluster.local",
				},
				secrets.NameForMTLSInbound: nil,
			},
		},
		{
			name: "requested certs nil returns all certs",
			serviceIdentitiesForService: map[service.MeshService][]identity.ServiceIdentity{
				{Name: "svc-1", Namespace: "ns-1"}: {
					identity.New("sa-1", "ns-1"),
					identity.New("sa-2", "ns-1"),
				},
				{Name: "svc-A", Namespace: "ns-A"}: {
					identity.New("sa-A", "ns-A"),
				},
			},
			expectedCertToSAN: map[string][]string{
				secrets.NameForUpstreamService("svc-1", "ns-1"): {
					"sa-1.ns-1.cluster.local",
					"sa-2.ns-1.cluster.local",
				},
				secrets.NameForUpstreamService("svc-A", "ns-A"): {
					"sa-A.ns-A.cluster.local",
				},
				secrets.NameForIdentity(proxySvcID): nil,
				secrets.NameForMTLSInbound:          nil,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// The Common Name of the xDS Certificate (issued to the Envoy on the Pod by the Injector) will
			// have be prefixed with the ID of the pod. It is the first chunk of a dot-separated string.
			proxy := envoy.NewProxy(envoy.KindSidecar, uuid.New(), proxySvcID, nil, 1)
			meshCatalog := catalog.NewMockMeshCataloger(mockCtrl)

			var services []service.MeshService
			for svc, identities := range tc.serviceIdentitiesForService {
				services = append(services, svc)
				meshCatalog.EXPECT().ListServiceIdentitiesForService(svc).Return(identities)
			}
			meshCatalog.EXPECT().ListOutboundServicesForIdentity(proxy.Identity).Return(services)

			var request *xds_discovery.DiscoveryRequest
			// Keep it as nil if no certs are requested.
			if tc.requestedCerts != nil {
				request = &xds_discovery.DiscoveryRequest{
					TypeUrl:       string(envoy.TypeSDS),
					ResourceNames: tc.requestedCerts,
				}
			}

			// ----- Test with an properly configured proxy
			resources, err := NewResponse(meshCatalog, proxy, request, certManager, nil)
			assert.Equal(err, nil, fmt.Sprintf("Error evaluating sds.NewResponse(): %s", err))
			assert.NotNil(resources)
			var certNames, expectedCertNames []string

			// Collecting cert names for the assert has an easier to read print statement on failure, compared to
			// the assert.Equal statement or assert.Len, which will print either nothing or the entire cert object respectively.
			for name := range tc.expectedCertToSAN {
				expectedCertNames = append(expectedCertNames, name)
			}

			for _, resource := range resources {
				secret, ok := resource.(*xds_auth.Secret)
				assert.True(ok)
				certNames = append(certNames, secret.Name)

				assert.Contains(tc.expectedCertToSAN, secret.Name)
				if len(tc.expectedCertToSAN[secret.Name]) == 0 {
					continue // nothing more to do.
				}
				assert.Len(secret.GetValidationContext().MatchTypedSubjectAltNames, len(tc.expectedCertToSAN[secret.Name]))
				for _, matchers := range secret.GetValidationContext().MatchTypedSubjectAltNames {
					assert.Contains(tc.expectedCertToSAN[secret.Name], matchers.Matcher.GetExact())
				}
			}
			assert.ElementsMatch(expectedCertNames, certNames)
		})
	}
}

func TestSDSBuilder(t *testing.T) {
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

		// list of certs requested of the form:
		// - "service-cert:namespace/service"
		// - "root-cert-for-mtls-inbound"
		// - "root-cert-for-mtls-outbound:namespace/service"
		requestedCerts []string

		// expectations
		expectedSANs        []string // only set for service-cert
		expectedSecretCount int
	}

	testCases := []testCase{
		// Test case 1: root-cert-for-mtls-inbound requested -------------------------------
		{
			name:            "test root-cert-for-mtls-inbound cert type request",
			serviceIdentity: identity.K8sServiceAccount{Name: "sa-1", Namespace: "ns-1"}.ToServiceIdentity(),
			requestedCerts:  []string{secrets.NameForMTLSInbound}, // root-cert requested

			// expectations
			expectedSANs:        nil,
			expectedSecretCount: 1,
		},
		// Test case 1 end -------------------------------

		// Test case 2: root-cert-for-mtls-outbound requested -------------------------------
		{
			name:            "test root-cert-for-mtls-outbound cert type request",
			serviceIdentity: identity.New("sa-1", "ns-1"),
			serviceIdentitiesForService: map[service.MeshService][]identity.ServiceIdentity{
				{
					Name:      "service-2",
					Namespace: "ns-2",
				}: {
					identity.New("sa-2", "ns-2"),
					identity.New("sa-3", "ns-2"),
				},
			},
			requestedCerts: []string{"root-cert-for-mtls-outbound:ns-2/service-2"}, // root-cert requested

			// expectations
			expectedSANs:        []string{"sa-2.ns-2.cluster.local", "sa-3.ns-2.cluster.local"},
			expectedSecretCount: 1,
		},
		// Test case 2 end -------------------------------

		// Test case 3: service-cert requested -------------------------------
		{
			name:            "test service-cert cert type request",
			serviceIdentity: identity.New("sa-1", "ns-1"),
			requestedCerts:  []string{"service-cert:ns-1/sa-1"}, // service-cert requested

			// expectations
			expectedSANs:        []string{},
			expectedSecretCount: 1,
		},
		// Test case 3 end -------------------------------

		// Test case 2: invalid requested -------------------------------
		{
			name:            "test invalid cert type request",
			serviceIdentity: identity.New("sa-1", "ns-1"),
			serviceIdentitiesForService: map[service.MeshService][]identity.ServiceIdentity{
				{
					Name:      "service-2",
					Namespace: "ns-2",
				}: {
					identity.New("sa-2", "ns-2"),
					identity.New("sa-3", "ns-2"),
				},
			},
			requestedCerts:      []string{"invalid:ns-2/service-2"}, // root-cert requested
			expectedSecretCount: 0,
		},
		// Test case 2 end -------------------------------
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("Testing test case %d: %s", i, tc.name), func(t *testing.T) {
			builder := NewBuilder()
			if tc.requestedCerts != nil {
				builder.SetRequestedCerts(tc.requestedCerts)
			}

			proxy := envoy.NewProxy(envoy.KindSidecar, uuid.New(), identity.New("sa-1", "ns-1"), nil, 1)
			builder.SetProxy(proxy).SetProxyCert(cert).SetTrustDomain("cluster.local")

			builder.SetServiceIdentitiesForService(tc.serviceIdentitiesForService)

			sdsSecrets := builder.Build()
			assert.Len(sdsSecrets, tc.expectedSecretCount)

			if tc.expectedSecretCount <= 0 {
				// nothing to validate further
				return
			}

			sdsSecret := sdsSecrets[0]

			switch sdsSecret.Name {
			case secrets.NameForMTLSInbound:
				assert.NotNil(sdsSecret.GetValidationContext().GetTrustedCa().GetInlineBytes())
			case secrets.NameForIdentity(tc.serviceIdentity):
				assert.NotNil(sdsSecret.GetTlsCertificate().GetCertificateChain().GetInlineBytes())
				assert.NotNil(sdsSecret.GetTlsCertificate().GetPrivateKey().GetInlineBytes())
			default:
				// outbound cert:
				actualSANs := subjectAltNamesToStr(sdsSecret.GetValidationContext().GetMatchTypedSubjectAltNames())
				assert.ElementsMatch(actualSANs, tc.expectedSANs)

				// Check trusted CA
				assert.NotNil(sdsSecret.GetValidationContext().GetTrustedCa().GetInlineBytes())
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
