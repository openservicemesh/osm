package sds

import (
	"fmt"
	"testing"

	xds_discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	xds_matcher "github.com/envoyproxy/go-control-plane/envoy/type/matcher/v3"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	tassert "github.com/stretchr/testify/assert"
	testclient "k8s.io/client-go/kubernetes/fake"

	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/certificate/providers/tresor"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/service"
)

// TestNewResponse sets up a fake kube client, then a pod and makes an SDS request,
// and finally verifies the response from sds.NewResponse().
func TestNewResponse(t *testing.T) {
	assert := tassert.New(t)

	// Setup a fake Kube client. We use this to create a full simulation of creating a pod with
	// the required xDS Certificate, properly formatted CommonName etc.
	fakeKubeClient := testclient.NewSimpleClientset()

	// We deliberately set the namespace and service accounts to random values
	// to ensure no hard-coded values sneak in.
	namespace := uuid.New().String()
	serviceAccount := uuid.New().String()

	// This is the thing we are going to be requesting (pretending that the Envoy is requesting it)
	request := &xds_discovery.DiscoveryRequest{
		TypeUrl: string(envoy.TypeSDS),
		ResourceNames: []string{
			envoy.SDSCert{Name: serviceAccount, CertType: envoy.ServiceCertType}.String(),
			envoy.SDSCert{Name: serviceAccount, CertType: envoy.RootCertTypeForMTLSInbound}.String(),
			envoy.SDSCert{Name: serviceAccount, CertType: envoy.RootCertTypeForHTTPS}.String(),
		},
	}

	stop := make(chan struct{})
	defer close(stop)

	// The Common Name of the xDS Certificate (issued to the Envoy on the Pod by the Injector) will
	// have be prefixed with the ID of the pod. It is the first chunk of a dot-separated string.
	podID := uuid.New().String()

	certCommonName := certificate.CommonName(fmt.Sprintf("%s.%s.%s", podID, serviceAccount, namespace))
	certSerialNumber := certificate.SerialNumber("123456")
	goodProxy := envoy.NewProxy(certCommonName, certSerialNumber, nil)

	badProxy := envoy.NewProxy("-certificate-common-name-is-invalid-", "-cert-serial-number-is-invalid-", nil)

	cfg := configurator.NewConfigurator(fakeKubeClient, stop, namespace, "-the-config-map-name-")
	certManager := tresor.NewFakeCertManager(cfg)
	meshCatalog := catalog.NewFakeMeshCatalog(fakeKubeClient)

	// ----- Test with a rogue proxy (does not belong to the mesh)
	actualSDSResponse, err := NewResponse(meshCatalog, badProxy, request, cfg, certManager)
	assert.Equal(err, catalog.ErrInvalidCertificateCN, "Expected a different error!")
	assert.Nil(actualSDSResponse)

	// ----- Test with an properly configured proxy
	actualSDSResponse, err = NewResponse(meshCatalog, goodProxy, request, cfg, certManager)
	assert.Equal(err, nil, fmt.Sprintf("Error evaluating sds.NewResponse(): %s", err))
	assert.NotNil(actualSDSResponse)
	assert.Equal(len(actualSDSResponse.Resources), 3)
	assert.Equal(actualSDSResponse.TypeUrl, "type.googleapis.com/envoy.extensions.transport_sockets.tls.v3.Secret")
}

func TestGetRootCert(t *testing.T) {
	assert := tassert.New(t)
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	// This is used to dynamically set expectations for each test in the list of table driven tests
	type dynamicMock struct {
		mockCatalog      *catalog.MockMeshCataloger
		mockConfigurator *configurator.MockConfigurator
		mockCertificater *certificate.MockCertificater
	}

	mockCertManager := certificate.NewMockManager(mockCtrl)

	type testCase struct {
		name            string
		sdsCert         envoy.SDSCert
		proxySvcAccount service.K8sServiceAccount
		prepare         func(d *dynamicMock)

		// expectations
		expectedSANs []string
		expectError  bool
	}

	testCases := []testCase{
		// Test case 1: tests SDS secret for inbound TLS secret -------------------------------
		{
			name: "test inbound MTLS certificate validation",
			sdsCert: envoy.SDSCert{
				Name:     "ns-1/service-1",
				CertType: envoy.RootCertTypeForMTLSInbound,
			},
			proxySvcAccount: service.K8sServiceAccount{Name: "sa-1", Namespace: "ns-1"},

			prepare: func(d *dynamicMock) {
				d.mockConfigurator.EXPECT().IsPermissiveTrafficPolicyMode().Return(false).Times(1)
				allowedInboundSvcAccounts := []service.K8sServiceAccount{
					{Name: "sa-2", Namespace: "ns-2"},
					{Name: "sa-3", Namespace: "ns-3"},
				}
				d.mockCatalog.EXPECT().ListAllowedInboundServiceAccounts(service.K8sServiceAccount{Name: "sa-1", Namespace: "ns-1"}).Return(allowedInboundSvcAccounts, nil).Times(1)
				d.mockCertificater.EXPECT().GetIssuingCA().Return([]byte("foo")).Times(1)
			},

			// expectations
			expectedSANs: []string{"sa-2.ns-2.cluster.local", "sa-3.ns-3.cluster.local"},
			expectError:  false,
		},
		// Test case 1 end -------------------------------

		// Test case 2: tests SDS secret for outbound TLS secret -------------------------------
		{
			name: "test outbound MTLS certificate validation",
			sdsCert: envoy.SDSCert{
				Name:     "ns-2/service-2",
				CertType: envoy.RootCertTypeForMTLSOutbound,
			},
			proxySvcAccount: service.K8sServiceAccount{Name: "sa-1", Namespace: "ns-1"},

			prepare: func(d *dynamicMock) {
				d.mockConfigurator.EXPECT().IsPermissiveTrafficPolicyMode().Return(false).Times(1)
				associatedSvcAccounts := []service.K8sServiceAccount{
					{Name: "sa-2", Namespace: "ns-2"},
					{Name: "sa-3", Namespace: "ns-2"},
				}
				d.mockCatalog.EXPECT().ListServiceAccountsForService(service.MeshService{Name: "service-2", Namespace: "ns-2"}).Return(associatedSvcAccounts, nil).Times(1)
				d.mockCertificater.EXPECT().GetIssuingCA().Return([]byte("foo")).Times(1)
			},

			// expectations
			expectedSANs: []string{"sa-2.ns-2.cluster.local", "sa-3.ns-2.cluster.local"},
			expectError:  false,
		},
		// Test case 2 end -------------------------------

		// Test case 3: tests SDS secret for permissive mode -------------------------------
		{
			name: "test permissive mode certificate validation",
			sdsCert: envoy.SDSCert{
				Name:     "ns-2/service-2",
				CertType: envoy.RootCertTypeForMTLSOutbound,
			},
			proxySvcAccount: service.K8sServiceAccount{Name: "sa-1", Namespace: "ns-1"},

			prepare: func(d *dynamicMock) {
				d.mockConfigurator.EXPECT().IsPermissiveTrafficPolicyMode().Return(true).Times(1)
				d.mockCertificater.EXPECT().GetIssuingCA().Return([]byte("foo")).Times(1)
			},

			// expectations
			expectedSANs: []string{}, // no SAN matching in permissive mode
			expectError:  false,
		},
		// Test case 3 end -------------------------------
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("Testing test case %d: %s", i, tc.name), func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			// Initialize the dynamic mocks
			d := dynamicMock{
				mockCatalog:      catalog.NewMockMeshCataloger(mockCtrl),
				mockConfigurator: configurator.NewMockConfigurator(mockCtrl),
				mockCertificater: certificate.NewMockCertificater(mockCtrl),
			}

			// Prepare the dynamic mock expectations for each test case
			if tc.prepare != nil {
				tc.prepare(&d)
			}

			s := &sdsImpl{
				svcAccount:  tc.proxySvcAccount,
				certManager: mockCertManager,

				// these points to the dynamic mocks which gets updated for each test
				meshCatalog: d.mockCatalog,
				cfg:         d.mockConfigurator,
			}

			// test the function
			sdsSecret, err := s.getRootCert(d.mockCertificater, tc.sdsCert)
			assert.Equal(err != nil, tc.expectError)

			actualSANs := subjectAltNamesToStr(sdsSecret.GetValidationContext().GetMatchSubjectAltNames())
			assert.ElementsMatch(actualSANs, tc.expectedSANs)
		})
	}
}

func TestGetServiceCert(t *testing.T) {
	assert := tassert.New(t)
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockCertificater := certificate.NewMockCertificater(mockCtrl)

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
			// Mock cert
			mockCertificater.EXPECT().GetCertificateChain().Return(tc.certChain).Times(1)
			mockCertificater.EXPECT().GetPrivateKey().Return(tc.privKey).Times(1)

			// Test the function
			sdsSecret, err := getServiceCertSecret(mockCertificater, tc.certName)

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

	mockCertManager := certificate.NewMockManager(mockCtrl)

	// This is used to dynamically set expectations for each test in the list of table driven tests
	type dynamicMock struct {
		mockCatalog      *catalog.MockMeshCataloger
		mockConfigurator *configurator.MockConfigurator
		mockCertificater *certificate.MockCertificater
	}

	type testCase struct {
		name            string
		proxySvcAccount service.K8sServiceAccount
		prepare         func(d *dynamicMock)

		// sdsCertType must match the requested cert type. used by the test for business logic
		sdsCertType envoy.SDSCertType
		// list of certs requested of the form:
		// - "service-cert:namespace/service"
		// - "root-cert-for-mtls-outbound:namespace/service"
		// - "root-cert-for-mtls-inbound:namespace/service"
		// - "root-cert-for-https:namespace/service"
		requestedCerts []string

		// expectations
		expectedSANs        []string // only set for service-cert
		expectedSecretCount int
	}

	testCases := []testCase{
		// Test case 1: root-cert-for-mtls-inbound requested -------------------------------
		{
			name:            "test root-cert-for-mtls-inbound cert type request",
			proxySvcAccount: service.K8sServiceAccount{Name: "sa-1", Namespace: "ns-1"},

			prepare: func(d *dynamicMock) {
				d.mockConfigurator.EXPECT().IsPermissiveTrafficPolicyMode().Return(false).Times(1)
				allowedInboundSvcAccounts := []service.K8sServiceAccount{
					{Name: "sa-2", Namespace: "ns-2"},
					{Name: "sa-3", Namespace: "ns-3"},
				}
				d.mockCatalog.EXPECT().ListAllowedInboundServiceAccounts(service.K8sServiceAccount{Name: "sa-1", Namespace: "ns-1"}).Return(allowedInboundSvcAccounts, nil).Times(1)
				d.mockCertificater.EXPECT().GetIssuingCA().Return([]byte("foo")).Times(1)
			},

			sdsCertType:    envoy.RootCertTypeForMTLSInbound,
			requestedCerts: []string{"root-cert-for-mtls-inbound:ns-1/service-1"}, // root-cert requested

			// expectations
			expectedSANs:        []string{"sa-2.ns-2.cluster.local", "sa-3.ns-3.cluster.local"},
			expectedSecretCount: 1,
		},
		// Test case 1 end -------------------------------

		// Test case 2: root-cert-for-mtls-outbound requested -------------------------------
		{
			name:            "test root-cert-for-mtls-outbound cert type request",
			proxySvcAccount: service.K8sServiceAccount{Name: "sa-1", Namespace: "ns-1"},

			prepare: func(d *dynamicMock) {
				d.mockConfigurator.EXPECT().IsPermissiveTrafficPolicyMode().Return(false).Times(1)
				associatedSvcAccounts := []service.K8sServiceAccount{
					{Name: "sa-2", Namespace: "ns-2"},
					{Name: "sa-3", Namespace: "ns-2"},
				}
				d.mockCatalog.EXPECT().ListServiceAccountsForService(
					service.MeshService{Name: "service-2", Namespace: "ns-2"}).Return(associatedSvcAccounts, nil).Times(1)
				d.mockCertificater.EXPECT().GetIssuingCA().Return([]byte("foo")).Times(1)
			},

			sdsCertType:    envoy.RootCertTypeForMTLSOutbound,
			requestedCerts: []string{"root-cert-for-mtls-outbound:ns-2/service-2"}, // root-cert requested

			// expectations
			expectedSANs:        []string{"sa-2.ns-2.cluster.local", "sa-3.ns-2.cluster.local"},
			expectedSecretCount: 1,
		},
		// Test case 2 end -------------------------------

		// Test case 3: root-cert-for-https requested -------------------------------
		{
			name:            "test root-cert-https cert type request",
			proxySvcAccount: service.K8sServiceAccount{Name: "sa-1", Namespace: "ns-1"},

			prepare: func(d *dynamicMock) {
				d.mockConfigurator.EXPECT().IsPermissiveTrafficPolicyMode().Return(false).Times(1)
				d.mockCertificater.EXPECT().GetIssuingCA().Return([]byte("foo")).Times(1)
			},

			sdsCertType:    envoy.RootCertTypeForHTTPS,
			requestedCerts: []string{"root-cert-https:ns-1/service-1"}, // root-cert requested

			// expectations
			expectedSANs:        []string{},
			expectedSecretCount: 1,
		},
		// Test case 3 end -------------------------------

		// Test case 4: service-cert requested -------------------------------
		{
			name:            "test root-cert-https cert type request",
			proxySvcAccount: service.K8sServiceAccount{Name: "sa-1", Namespace: "ns-1"},

			prepare: func(d *dynamicMock) {
				d.mockCertificater.EXPECT().GetCertificateChain().Return([]byte("foo")).Times(1)
				d.mockCertificater.EXPECT().GetPrivateKey().Return([]byte("foo")).Times(1)
			},

			sdsCertType:    envoy.ServiceCertType,
			requestedCerts: []string{"service-cert:ns-1/service-1"}, // service-cert requested

			// expectations
			expectedSANs:        []string{},
			expectedSecretCount: 1,
		},
		// Test case 4 end -------------------------------

		// Test case 5: invalid cert type requested -------------------------------
		{
			name:            "test root-cert-https cert type request",
			proxySvcAccount: service.K8sServiceAccount{Name: "sa-1", Namespace: "ns-1"},

			prepare: nil,

			sdsCertType:    envoy.SDSCertType("invalid"),
			requestedCerts: []string{"invalid:ns-1/service-1"}, // service-cert requested

			// expectations
			expectedSANs:        []string{},
			expectedSecretCount: 0, // error is logged and no SDS secret is created
		},
		// Test case 5 end -------------------------------
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("Testing test case %d: %s", i, tc.name), func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			// Initialize the dynamic mocks
			d := dynamicMock{
				mockCatalog:      catalog.NewMockMeshCataloger(mockCtrl),
				mockConfigurator: configurator.NewMockConfigurator(mockCtrl),
				mockCertificater: certificate.NewMockCertificater(mockCtrl),
			}

			// Prepare the dynamic mock expectations for each test case
			if tc.prepare != nil {
				tc.prepare(&d)
			}

			certCommonName := certificate.CommonName(fmt.Sprintf("%s.%s.%s", uuid.New().String(), "sa-1", "ns-1"))
			certSerialNumber := certificate.SerialNumber("123456")
			s := &sdsImpl{
				svcAccount:  tc.proxySvcAccount,
				certManager: mockCertManager,

				// these points to the dynamic mocks which gets updated for each test
				meshCatalog: d.mockCatalog,
				cfg:         d.mockConfigurator,
			}

			proxy := envoy.NewProxy(certCommonName, certSerialNumber, nil)

			// test the function
			sdsSecrets := s.getSDSSecrets(d.mockCertificater, tc.requestedCerts, proxy)
			assert.Len(sdsSecrets, tc.expectedSecretCount)

			if tc.expectedSecretCount <= 0 {
				// nothing to validate further
				return
			}

			sdsSecret := sdsSecrets[0]

			// verify different cert types
			switch tc.sdsCertType {
			// Verify SAN for inbound and outbound MTLS certs
			case envoy.RootCertTypeForMTLSInbound, envoy.RootCertTypeForMTLSOutbound, envoy.RootCertTypeForHTTPS:
				// Check SANs
				actualSANs := subjectAltNamesToStr(sdsSecret.GetValidationContext().GetMatchSubjectAltNames())
				assert.ElementsMatch(actualSANs, tc.expectedSANs)

				// Check trusted CA
				assert.NotNil(sdsSecret.GetValidationContext().GetTrustedCa().GetInlineBytes())

			case envoy.ServiceCertType:
				assert.NotNil(sdsSecret.GetTlsCertificate().GetCertificateChain().GetInlineBytes())
				assert.NotNil(sdsSecret.GetTlsCertificate().GetPrivateKey().GetInlineBytes())
			}
		})
	}
}

func TestGetSubjectAltNamesFromSvcAccount(t *testing.T) {
	assert := tassert.New(t)

	type testCase struct {
		svcAccounts         []service.K8sServiceAccount
		expectedSANMatchers []*xds_matcher.StringMatcher
	}

	testCases := []testCase{
		{
			svcAccounts: []service.K8sServiceAccount{
				{Name: "sa-1", Namespace: "ns-1"},
				{Name: "sa-2", Namespace: "ns-2"},
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
			actual := getSubjectAltNamesFromSvcAccount(tc.svcAccounts)
			assert.ElementsMatch(actual, tc.expectedSANMatchers)
		})
	}
}

func TestSubjectAltNamesToStr(t *testing.T) {
	assert := tassert.New(t)

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
			actual := subjectAltNamesToStr(tc.sanMatchers)
			assert.ElementsMatch(actual, tc.strSANs)
		})
	}
}
