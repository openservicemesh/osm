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
	testclient "k8s.io/client-go/kubernetes/fake"

	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/envoy/secrets"
	configFake "github.com/openservicemesh/osm/pkg/gen/client/config/clientset/versioned/fake"
	"github.com/openservicemesh/osm/pkg/k8s/informers"

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

	// Setup a fake Kube client. We use this to create a full simulation of creating a pod with
	// the required xDS Certificate, properly formatted CommonName etc.
	fakeKubeClient := testclient.NewSimpleClientset()
	fakeConfigClient := configFake.NewSimpleClientset()

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

	proxy := envoy.NewProxy(envoy.KindSidecar, podID, proxySvcAccount.ToServiceIdentity(), nil)
	ic, err := informers.NewInformerCollection("osm", stop, informers.WithKubeClient(fakeKubeClient), informers.WithConfigClient(fakeConfigClient, "-the-mesh-config-name-", "-osm-namespace-"))
	assert.Nil(err)

	cfg := configurator.NewConfigurator(ic, "-osm-namespace-", "-the-mesh-config-name-", nil)
	certManager := tresorFake.NewFake(nil, 1*time.Hour)
	meshCatalog := catalogFake.NewFakeMeshCatalog(fakeKubeClient, fakeConfigClient)

	// ----- Test with an properly configured proxy
	resources, err := NewResponse(meshCatalog, proxy, request, cfg, certManager, nil)
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

	// This is used to dynamically set expectations for each test in the list of table driven tests
	type dynamicMock struct {
		mockCatalog *catalog.MockMeshCataloger
	}

	type testCase struct {
		name            string
		sdsCert         secrets.SDSCert
		serviceIdentity identity.ServiceIdentity
		prepare         func(d *dynamicMock)

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

			prepare: func(d *dynamicMock) {},

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

			prepare: func(d *dynamicMock) {
				associatedSvcAccounts := []identity.ServiceIdentity{
					identity.K8sServiceAccount{Name: "sa-2", Namespace: "ns-2"}.ToServiceIdentity(),
					identity.K8sServiceAccount{Name: "sa-3", Namespace: "ns-2"}.ToServiceIdentity(),
				}
				d.mockCatalog.EXPECT().ListServiceIdentitiesForService(service.MeshService{
					Name:      "service-2",
					Namespace: "ns-2",
				}).Return(associatedSvcAccounts).Times(1)
			},

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
			fakeCertManager, err := certificate.FakeCertManager()
			if err != nil {
				t.Error(err)
			}

			// Initialize the dynamic mocks
			d := dynamicMock{
				mockCatalog: catalog.NewMockMeshCataloger(mockCtrl),
			}

			// Prepare the dynamic mock expectations for each test case
			if tc.prepare != nil {
				tc.prepare(&d)
			}

			s := &sdsImpl{
				serviceIdentity: tc.serviceIdentity,
				certManager:     fakeCertManager,

				// these points to the dynamic mocks which gets updated for each test
				meshCatalog: d.mockCatalog,
			}

			// test the function
			sdsSecret, err := s.getRootCert(cert, tc.sdsCert)
			assert.Equal(err != nil, tc.expectError)

			if err != nil {
				actualSANs := subjectAltNamesToStr(sdsSecret.GetValidationContext().GetMatchSubjectAltNames())
				assert.ElementsMatch(actualSANs, tc.expectedSANs)
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

			// Test the function
			sdsSecret, err := getServiceCertSecret(cert, tc.certName)

			assert.Equal(err != nil, tc.expectError)
			assert.NotNil(sdsSecret)
			assert.Equal(sdsSecret.GetTlsCertificate().GetCertificateChain().GetInlineBytes(), tc.certChain)
			assert.Equal(sdsSecret.GetTlsCertificate().GetPrivateKey().GetInlineBytes(), tc.privKey)
		})
	}
}

func TestGetSDSSecrets(t *testing.T) {
	assert := tassert.New(t)
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	fakeCertManager, err := certificate.FakeCertManager()
	if err != nil {
		t.Error(err)
	}

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

			s := &sdsImpl{
				serviceIdentity: tc.serviceIdentity,
				certManager:     fakeCertManager,

				// these points to the dynamic mocks which gets updated for each test
				meshCatalog: d.mockCatalog,
				TrustDomain: "cluster.local",
			}

			proxy := envoy.NewProxy(envoy.KindSidecar, uuid.New(), identity.New("sa-1", "ns-1"), nil)

			// test the function
			sdsSecrets := s.getSDSSecrets(cert, tc.requestedCerts, proxy)
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
				actualSANs := subjectAltNamesToStr(sdsSecret.GetValidationContext().GetMatchSubjectAltNames())
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

func TestGetSubjectAltNamesFromSvcAccount(t *testing.T) {
	type testCase struct {
		serviceIdentities   []identity.ServiceIdentity
		expectedSANMatchers []*xds_matcher.StringMatcher
	}

	testCases := []testCase{
		{
			serviceIdentities: []identity.ServiceIdentity{
				identity.K8sServiceAccount{Name: "sa-1", Namespace: "ns-1"}.ToServiceIdentity(),
				identity.K8sServiceAccount{Name: "sa-2", Namespace: "ns-2"}.ToServiceIdentity(),
			},
			expectedSANMatchers: []*xds_matcher.StringMatcher{
				{
					MatchPattern: &xds_matcher.StringMatcher_Exact{
						Exact: "sa-1.ns-1.cluster.local",
					},
				},
				{
					MatchPattern: &xds_matcher.StringMatcher_Exact{
						Exact: "sa-2.ns-2.cluster.local",
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
		sanMatchers []*xds_matcher.StringMatcher
		strSANs     []string
	}

	testCases := []testCase{
		{
			sanMatchers: []*xds_matcher.StringMatcher{
				{
					MatchPattern: &xds_matcher.StringMatcher_Exact{
						Exact: "sa-1.ns-1.cluster.local",
					},
				},
				{
					MatchPattern: &xds_matcher.StringMatcher_Exact{
						Exact: "sa-2.ns-2.cluster.local",
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
