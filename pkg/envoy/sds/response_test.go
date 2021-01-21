package sds

import (
	"fmt"
	"testing"

	xds_matcher "github.com/envoyproxy/go-control-plane/envoy/type/matcher/v3"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	tassert "github.com/stretchr/testify/assert"

	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/service"
)

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

	testCases := []struct {
		name            string
		sdsCert         envoy.SDSCert
		proxyService    service.MeshService
		proxySvcAccount service.K8sServiceAccount
		prepare         func(d *dynamicMock)

		// expectations
		expectedSANs []string
		expectError  bool
	}{
		// Test case 1: tests SDS secret for inbound TLS secret -------------------------------
		{
			name: "test inbound MTLS certificate validation",
			sdsCert: envoy.SDSCert{
				MeshService: service.MeshService{Name: "service-1", Namespace: "ns-1"},
				CertType:    envoy.RootCertTypeForMTLSInbound,
			},
			proxyService:    service.MeshService{Name: "service-1", Namespace: "ns-1"},
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
				MeshService: service.MeshService{Name: "service-2", Namespace: "ns-2"},
				CertType:    envoy.RootCertTypeForMTLSOutbound,
			},
			proxyService:    service.MeshService{Name: "service-1", Namespace: "ns-1"},
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
				MeshService: service.MeshService{Name: "service-2", Namespace: "ns-2"},
				CertType:    envoy.RootCertTypeForMTLSOutbound,
			},
			proxyService:    service.MeshService{Name: "service-1", Namespace: "ns-1"},
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

			certCommonName := certificate.CommonName(fmt.Sprintf("%s.%s.%s", uuid.New().String(), "sa-1", "ns-1"))
			certSerialNumber := certificate.SerialNumber("123456")
			s := &sdsImpl{
				proxyServices: []service.MeshService{tc.proxyService},
				svcAccount:    tc.proxySvcAccount,
				proxy:         envoy.NewProxy(certCommonName, certSerialNumber, nil),
				certManager:   mockCertManager,

				// these points to the dynamic mocks which gets updated for each test
				meshCatalog: d.mockCatalog,
				cfg:         d.mockConfigurator,
			}

			// test the function
			sdsSecret, err := s.getRootCert(d.mockCertificater, tc.sdsCert, tc.proxyService)
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

	testCases := []struct {
		certName    string
		certChain   []byte
		privKey     []byte
		expectError bool
	}{
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

	testCases := []struct {
		name            string
		proxyService    service.MeshService
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
	}{
		// Test case 1: root-cert-for-mtls-inbound requested -------------------------------
		{
			name:            "test root-cert-for-mtls-inbound cert type request",
			proxyService:    service.MeshService{Name: "service-1", Namespace: "ns-1"},
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
			proxyService:    service.MeshService{Name: "service-1", Namespace: "ns-1"},
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
			proxyService:    service.MeshService{Name: "service-1", Namespace: "ns-1"},
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
			proxyService:    service.MeshService{Name: "service-1", Namespace: "ns-1"},
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
			proxyService:    service.MeshService{Name: "service-1", Namespace: "ns-1"},
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
				proxyServices: []service.MeshService{tc.proxyService},
				svcAccount:    tc.proxySvcAccount,
				proxy:         envoy.NewProxy(certCommonName, certSerialNumber, nil),
				certManager:   mockCertManager,

				// these points to the dynamic mocks which gets updated for each test
				meshCatalog: d.mockCatalog,
				cfg:         d.mockConfigurator,
			}

			// test the function
			sdsSecrets := s.getSDSSecrets(d.mockCertificater, tc.requestedCerts, tc.proxyService)
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

	testCases := []struct {
		svcAccounts         []service.K8sServiceAccount
		expectedSANMatchers []*xds_matcher.StringMatcher
	}{
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

	testCases := []struct {
		sanMatchers []*xds_matcher.StringMatcher
		strSANs     []string
	}{
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
