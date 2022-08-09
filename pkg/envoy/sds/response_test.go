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

	"github.com/openservicemesh/osm/pkg/envoy/secrets"

	"github.com/openservicemesh/osm/pkg/catalog"
	catalogFake "github.com/openservicemesh/osm/pkg/catalog/fake"
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

	// We deliberately set the namespace and service accounts to random values
	// to ensure no hard-coded values sneak in.
	proxySvcAccount := identity.K8sServiceAccount{Name: uuid.New().String(), Namespace: uuid.New().String()}

	// This is the thing we are going to be requesting (pretending that the Envoy is requesting it)
	request := &xds_discovery.DiscoveryRequest{
		TypeUrl: string(envoy.TypeSDS),
		ResourceNames: []string{
			secrets.SDSCert{Name: proxySvcAccount.String(), CertType: secrets.ServiceCertType}.String(),
			secrets.SDSCert{Name: proxySvcAccount.String(), CertType: secrets.RootCertTypeForMTLSInbound}.String(),
		},
	}

	stop := make(chan struct{})
	defer close(stop)

	// The Common Name of the xDS Certificate (issued to the Envoy on the Pod by the Injector) will
	// have be prefixed with the ID of the pod. It is the first chunk of a dot-separated string.
	podID := uuid.New()

	proxy := envoy.NewProxy(envoy.KindSidecar, podID, proxySvcAccount.ToServiceIdentity(), nil, 1)

	certManager := tresorFake.NewFake(1 * time.Hour)
	meshCatalog := catalogFake.NewFakeMeshCatalog(nil)

	// ----- Test with an properly configured proxy
	resources, err := NewResponse(meshCatalog, proxy, request, certManager, nil)
	assert.Equal(err, nil, fmt.Sprintf("Error evaluating sds.NewResponse(): %s", err))
	assert.NotNil(resources)
	assert.Equal(len(resources), 2) // 1. service-cert, 2. root-cert-for-mtls-inbound (refer to the DiscoveryRequest 'request')
	_, ok := resources[0].(*xds_auth.Secret)
	assert.True(ok)
}

func TestGetRootCert(t *testing.T) {
	assert := tassert.New(t)
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	type testCase struct {
		name            string
		sdsCert         secrets.SDSCert
		serviceIdentity identity.ServiceIdentity

		// expectations
		expectedSANs []string
		expectError  bool
	}

	testCases := []testCase{
		// Test case 1: tests SDS secret for inbound TLS secret -------------------------------
		{
			name: "test inbound MTLS certificate validation",
			sdsCert: secrets.SDSCert{
				Name:     "ns-1/sa-1",
				CertType: secrets.RootCertTypeForMTLSInbound,
			},
			serviceIdentity: identity.K8sServiceAccount{Name: "sa-1", Namespace: "ns-1"}.ToServiceIdentity(),

			// expectations
			expectedSANs: nil,
			expectError:  false,
		},
		// Test case 1 end -------------------------------

		// Test case 2: tests SDS secret for outbound TLS secret -------------------------------
		{
			name: "test outbound MTLS certificate validation",
			sdsCert: secrets.SDSCert{
				Name:     "ns-2/service-2",
				CertType: secrets.RootCertTypeForMTLSOutbound,
			},
			serviceIdentity: identity.K8sServiceAccount{Name: "sa-1", Namespace: "ns-1"}.ToServiceIdentity(),

			// expectations
			expectedSANs: []string{"sa-2.ns-2.cluster.local", "sa-3.ns-2.cluster.local"},
			expectError:  false,
		},
		// Test case 2 end -------------------------------
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("Testing test case %d: %s", i, tc.name), func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			cert := &certificate.Certificate{}

			builder := NewBuilder()
			builder.SetProxyCert(cert)

			meshService, err := tc.sdsCert.GetMeshService()
			assert.Equal(err, nil, fmt.Sprintf("Error retrieving mesh service: %s", err))

			serviceIdentitiesForServices := make(map[service.MeshService][]identity.ServiceIdentity)
			serviceIdentitiesForServices[*meshService] = []identity.ServiceIdentity{tc.serviceIdentity}
			builder.SetServiceIdentitiesForService(serviceIdentitiesForServices)

			// test the function
			sdsSecret, err := builder.buildRootCertSecret(tc.sdsCert)
			assert.Equal(err != nil, tc.expectError)

			if err != nil {
				actualSANs := subjectAltNamesToStr(sdsSecret.GetValidationContext().GetMatchTypedSubjectAltNames())
				assert.ElementsMatch(actualSANs, tc.expectedSANs)
				// Check trusted CA
				assert.NotNil(sdsSecret.GetValidationContext().GetTrustedCa().GetInlineBytes())
			}
		})
	}
}

func TestGetServiceCert(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	type testCase struct {
		certName    string
		certChain   []byte
		privKey     []byte
		expectError bool
	}

	testCases := []testCase{
		{"foo", []byte("cert-chain"), []byte("priv-key"), false},
		{"bar", []byte("cert-chain-2"), []byte("priv-key-2"), false},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("Testing test case %d", i), func(t *testing.T) {
			assert := tassert.New(t)

			// Mock cert
			cert := &certificate.Certificate{
				CertChain:  tc.certChain,
				PrivateKey: tc.privKey,
			}

			builder := NewBuilder()
			builder.SetProxyCert(cert)
			// Test the function
			sdsSecret, err := builder.buildServiceCertSecret(tc.certName) // getServiceCertSecret(cert, tc.certName)

			assert.Equal(err != nil, tc.expectError)
			assert.NotNil(sdsSecret)
			assert.Equal(sdsSecret.GetTlsCertificate().GetCertificateChain().GetInlineBytes(), tc.certChain)
			assert.Equal(sdsSecret.GetTlsCertificate().GetPrivateKey().GetInlineBytes(), tc.privKey)
		})
	}
}

func TestSDSBuilder(t *testing.T) {
	assert := tassert.New(t)
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	cert := &certificate.Certificate{
		CertChain:  []byte("foo"),
		PrivateKey: []byte("foo"),
		IssuingCA:  []byte("foo"),
		TrustedCAs: []byte("foo"),
	}

	// This is used to dynamically set expectations for each test in the list of table driven tests
	type dynamicMock struct {
		mockCatalog *catalog.MockMeshCataloger
	}

	type testCase struct {
		name            string
		serviceIdentity identity.ServiceIdentity
		prepare         func(d *dynamicMock)

		// sdsCertType must match the requested cert type. used by the test for business logic
		sdsCertType secrets.SDSCertType
		// list of certs requested of the form:
		// - "service-cert:namespace/service"
		// - "root-cert-for-mtls-outbound:namespace/service"
		// - "root-cert-for-mtls-inbound:namespace/service"
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

			prepare: func(d *dynamicMock) {},

			sdsCertType:    secrets.RootCertTypeForMTLSInbound,
			requestedCerts: []string{"root-cert-for-mtls-inbound:ns-1/sa-1"}, // root-cert requested

			// expectations
			expectedSANs:        nil,
			expectedSecretCount: 1,
		},
		// Test case 1 end -------------------------------

		// Test case 2: root-cert-for-mtls-outbound requested -------------------------------
		{
			name:            "test root-cert-for-mtls-outbound cert type request",
			serviceIdentity: identity.K8sServiceAccount{Name: "sa-1", Namespace: "ns-1"}.ToServiceIdentity(),

			prepare: func(d *dynamicMock) {
				associatedSvcAccounts := []identity.ServiceIdentity{
					identity.K8sServiceAccount{Name: "sa-2", Namespace: "ns-2"}.ToServiceIdentity(),
					identity.K8sServiceAccount{Name: "sa-3", Namespace: "ns-2"}.ToServiceIdentity(),
				}
				svc := service.MeshService{
					Name:      "service-2",
					Namespace: "ns-2",
				}
				d.mockCatalog.EXPECT().ListServiceIdentitiesForService(svc).Return(associatedSvcAccounts).Times(1)
			},

			sdsCertType:    secrets.RootCertTypeForMTLSOutbound,
			requestedCerts: []string{"root-cert-for-mtls-outbound:ns-2/service-2"}, // root-cert requested

			// expectations
			expectedSANs:        []string{"sa-2.ns-2.cluster.local", "sa-3.ns-2.cluster.local"},
			expectedSecretCount: 1,
		},
		// Test case 2 end -------------------------------

		// Test case 3: service-cert requested -------------------------------
		{
			name:            "test service-cert cert type request",
			serviceIdentity: identity.K8sServiceAccount{Name: "sa-1", Namespace: "ns-1"}.ToServiceIdentity(),

			prepare: func(d *dynamicMock) {},

			sdsCertType:    secrets.ServiceCertType,
			requestedCerts: []string{"service-cert:ns-1/sa-1"}, // service-cert requested

			// expectations
			expectedSANs:        []string{},
			expectedSecretCount: 1,
		},
		// Test case 3 end -------------------------------

		// Test case 4: invalid cert type requested -------------------------------
		{
			name:            "test invalid cert type request",
			serviceIdentity: identity.K8sServiceAccount{Name: "sa-1", Namespace: "ns-1"}.ToServiceIdentity(),

			prepare: nil,

			sdsCertType:    secrets.SDSCertType("invalid"),
			requestedCerts: []string{"invalid:ns-1/service-1"}, // service-cert requested

			// expectations
			expectedSANs:        []string{},
			expectedSecretCount: 0, // error is logged and no SDS secret is created
		},
		// Test case 4 end -------------------------------
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("Testing test case %d: %s", i, tc.name), func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			// Initialize the dynamic mocks
			d := dynamicMock{
				mockCatalog: catalog.NewMockMeshCataloger(mockCtrl),
			}

			// Prepare the dynamic mock expectations for each test case
			if tc.prepare != nil {
				tc.prepare(&d)
			}

			builder := NewBuilder()
			builder.SetRequestedCerts(tc.requestedCerts)

			proxy := envoy.NewProxy(envoy.KindSidecar, uuid.New(), identity.New("sa-1", "ns-1"), nil, 1)
			builder.SetProxy(proxy).SetProxyCert(cert).SetTrustDomain("cluster.local")

			serviceIdentitiesForServices := getServiceIdentitiesForOutboundServices(tc.requestedCerts, d.mockCatalog)
			builder.SetServiceIdentitiesForService(serviceIdentitiesForServices)

			sdsSecrets := builder.Build()
			assert.Len(sdsSecrets, tc.expectedSecretCount)

			if tc.expectedSecretCount <= 0 {
				// nothing to validate further
				return
			}

			sdsSecret := sdsSecrets[0]

			// verify different cert types
			switch tc.sdsCertType {
			// Verify SAN for inbound and outbound MTLS certs
			case secrets.RootCertTypeForMTLSInbound, secrets.RootCertTypeForMTLSOutbound:
				// Check SANs
				actualSANs := subjectAltNamesToStr(sdsSecret.GetValidationContext().GetMatchTypedSubjectAltNames())
				assert.ElementsMatch(actualSANs, tc.expectedSANs)

				// Check trusted CA
				assert.NotNil(sdsSecret.GetValidationContext().GetTrustedCa().GetInlineBytes())

			case secrets.ServiceCertType:
				assert.NotNil(sdsSecret.GetTlsCertificate().GetCertificateChain().GetInlineBytes())
				assert.NotNil(sdsSecret.GetTlsCertificate().GetPrivateKey().GetInlineBytes())
			}
		})
	}
}

// Input requestedCerts []string, meshCatalog catalog.MeshCataloger
// output map[service.MeshService][]identity.ServiceIdentity
// confirm meshCatalog call for secrets.RootCertTypeForMTLSOutbound
func TestGetServiceIdentitiesForServices(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	proxySvcAccount := identity.K8sServiceAccount{Name: "sa-1", Namespace: "ns-1"}

	// This is used to dynamically set expectations for each test in the list of table driven tests
	type dynamicMock struct {
		mockCatalog *catalog.MockMeshCataloger
	}

	type testCase struct {
		name                      string
		requestedCerts            []string
		prepare                   func(d *dynamicMock)
		expectedService           service.MeshService
		expectedServiceIdentities []identity.ServiceIdentity
	}

	testCases := []testCase{
		// Test case 1: test service identities are retrieved for requested root certs for outbound MTLS -------------------------------
		{
			name:           "test service identities are retrieved for requested root certs for outbound MTLS",
			requestedCerts: []string{secrets.SDSCert{Name: proxySvcAccount.String(), CertType: secrets.RootCertTypeForMTLSOutbound}.String()},
			prepare: func(d *dynamicMock) {
				d.mockCatalog.EXPECT().ListServiceIdentitiesForService(service.MeshService{
					Name:      "sa-1",
					Namespace: "ns-1",
				}).Return([]identity.ServiceIdentity{
					identity.K8sServiceAccount{Name: "sa-1", Namespace: "ns-1"}.ToServiceIdentity(),
				}).Times(1)
			},
			expectedService:           service.MeshService{Name: "sa-1", Namespace: "ns-1"},
			expectedServiceIdentities: []identity.ServiceIdentity{identity.K8sServiceAccount{Name: "sa-1", Namespace: "ns-1"}.ToServiceIdentity()},
		},
		// Test case 1 end  -------------------------------

		// Test case 2: test service identities aren't retrieved for requested certs that aren't for outbound MTLS -------------------------------
		{
			requestedCerts:            []string{secrets.SDSCert{Name: proxySvcAccount.String(), CertType: secrets.ServiceCertType}.String(), secrets.SDSCert{Name: proxySvcAccount.String(), CertType: secrets.RootCertTypeForMTLSInbound}.String()},
			prepare:                   func(d *dynamicMock) {},
			expectedService:           service.MeshService{},
			expectedServiceIdentities: nil,
		},
		// Test case 2 end  -------------------------------
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("Testing test case %d: %s", i, tc.name), func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()
			assert := tassert.New(t)

			// Initialize the dynamic mocks
			d := dynamicMock{
				mockCatalog: catalog.NewMockMeshCataloger(mockCtrl),
			}

			// Prepare the dynamic mock expectations for each test case
			if tc.prepare != nil {
				tc.prepare(&d)
			}

			serviceIdentities := getServiceIdentitiesForOutboundServices(tc.requestedCerts, d.mockCatalog)
			if len(serviceIdentities) != 0 {
				assert.Equal(serviceIdentities[tc.expectedService], tc.expectedServiceIdentities)
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
