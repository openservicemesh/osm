package sds

import (
	"fmt"

	"github.com/google/uuid"

	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/service"
)

func TestGetRootCert(t *testing.T) {
	assert := assert.New(t)
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockCertificater := certificate.NewMockCertificater(mockCtrl)
	mockCertManager := certificate.NewMockManager(mockCtrl)
	mockCatalog := catalog.NewMockMeshCataloger(mockCtrl)
	mockConfigurator := configurator.NewMockConfigurator(mockCtrl)

	testCases := []struct {
		s                             *sdsImpl
		certCN                        certificate.CommonName
		sdsCert                       envoy.SDSCert
		proxyService                  service.MeshService
		allowedDirectionalSvcAccounts []service.K8sServiceAccount
		permissiveMode                bool

		// expectations
		expectedSANs []string
		expectError  bool
	}{
		// Test case 1: tests SDS secret for inbound TLS secret -------------------------------
		{
			s: &sdsImpl{
				proxyServices: []service.MeshService{
					{Name: "service-1", Namespace: "ns-1"},
				},
				svcAccount:  service.K8sServiceAccount{Name: "sa-1", Namespace: "ns-1"},
				proxy:       envoy.NewProxy(certificate.CommonName(fmt.Sprintf("%s.%s.%s", uuid.New().String(), "sa-1", "ns-1")), nil),
				meshCatalog: mockCatalog,
				certManager: mockCertManager,
				cfg:         mockConfigurator,
			},
			certCN: certificate.CommonName("sa-1.ns-1.cluster.local"),
			sdsCert: envoy.SDSCert{
				MeshService: service.MeshService{Name: "service-1", Namespace: "ns-1"},
				CertType:    envoy.RootCertTypeForMTLSInbound,
			},
			proxyService: service.MeshService{Name: "service-1", Namespace: "ns-1"},
			allowedDirectionalSvcAccounts: []service.K8sServiceAccount{
				{Name: "sa-2", Namespace: "ns-2"},
				{Name: "sa-3", Namespace: "ns-3"},
			},
			permissiveMode: false,

			// expectations
			expectedSANs: []string{"sa-2.ns-2.cluster.local", "sa-3.ns-3.cluster.local"},
			expectError:  false,
		},
		// Test case 1 end -------------------------------

		// Test case 2: tests SDS secret for outbound TLS secret -------------------------------
		{
			s: &sdsImpl{
				proxyServices: []service.MeshService{
					{Name: "service-1", Namespace: "ns-1"},
				},
				svcAccount:  service.K8sServiceAccount{Name: "sa-1", Namespace: "ns-1"},
				proxy:       envoy.NewProxy(certificate.CommonName(fmt.Sprintf("%s.%s.%s", uuid.New().String(), "sa-1", "ns-1")), nil),
				meshCatalog: mockCatalog,
				certManager: mockCertManager,
				cfg:         mockConfigurator,
			},
			certCN: certificate.CommonName("sa-1.ns-1.cluster.local"),
			sdsCert: envoy.SDSCert{
				MeshService: service.MeshService{Name: "service-1", Namespace: "ns-1"},
				CertType:    envoy.RootCertTypeForMTLSOutbound,
			},
			proxyService: service.MeshService{Name: "service-1", Namespace: "ns-1"},
			allowedDirectionalSvcAccounts: []service.K8sServiceAccount{
				{Name: "sa-2", Namespace: "ns-2"},
				{Name: "sa-3", Namespace: "ns-3"},
			},
			permissiveMode: false,

			// expectations
			expectedSANs: []string{"sa-2.ns-2.cluster.local", "sa-3.ns-3.cluster.local"},
			expectError:  false,
		},
		// Test case 2 end -------------------------------
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("Testing test case %d", i), func(t *testing.T) {
			// Mock catalog calls for tests
			mockConfigurator.EXPECT().IsPermissiveTrafficPolicyMode().Return(tc.permissiveMode).Times(1)
			if !tc.permissiveMode {
				if tc.sdsCert.CertType == envoy.RootCertTypeForMTLSOutbound {
					// outbound cert
					mockCatalog.EXPECT().ListAllowedOutboundServiceAccounts(tc.s.svcAccount).Return(tc.allowedDirectionalSvcAccounts, nil).Times(1)
				} else if tc.sdsCert.CertType == envoy.RootCertTypeForMTLSInbound {
					// inbound cert
					mockCatalog.EXPECT().ListAllowedInboundServiceAccounts(tc.s.svcAccount).Return(tc.allowedDirectionalSvcAccounts, nil).Times(1)
				}
			}

			// Mock CA cert
			mockCertificater.EXPECT().GetIssuingCA().Return([]byte("foo")).Times(1)

			// test the function
			sdsSecret, err := tc.s.getRootCert(mockCertificater, tc.sdsCert, tc.proxyService)

			// build the list of SAN from the returned secret
			var actualSANs []string
			for _, stringMatcher := range sdsSecret.GetValidationContext().GetMatchSubjectAltNames() {
				actualSANs = append(actualSANs, stringMatcher.GetExact())
			}

			assert.Equal(err != nil, tc.expectError)
			assert.ElementsMatch(actualSANs, tc.expectedSANs)
		})
	}
}

func TestGetServiceCert(t *testing.T) {
	assert := assert.New(t)
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
	assert := assert.New(t)
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockCertificater := certificate.NewMockCertificater(mockCtrl)
	mockCertManager := certificate.NewMockManager(mockCtrl)
	mockCatalog := catalog.NewMockMeshCataloger(mockCtrl)
	mockConfigurator := configurator.NewMockConfigurator(mockCtrl)

	testCases := []struct {
		s                             *sdsImpl
		certCN                        certificate.CommonName
		proxyService                  service.MeshService
		allowedDirectionalSvcAccounts []service.K8sServiceAccount
		permissiveMode                bool

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
			s: &sdsImpl{
				proxyServices: []service.MeshService{
					{Name: "service-1", Namespace: "ns-1"},
				},
				svcAccount:  service.K8sServiceAccount{Name: "sa-1", Namespace: "ns-1"},
				proxy:       envoy.NewProxy(certificate.CommonName(fmt.Sprintf("%s.%s.%s", uuid.New().String(), "sa-1", "ns-1")), nil),
				meshCatalog: mockCatalog,
				certManager: mockCertManager,
				cfg:         mockConfigurator,
			},
			certCN:       certificate.CommonName("sa-1.ns-1.cluster.local"),
			proxyService: service.MeshService{Name: "service-1", Namespace: "ns-1"},
			allowedDirectionalSvcAccounts: []service.K8sServiceAccount{
				{Name: "sa-2", Namespace: "ns-2"},
				{Name: "sa-3", Namespace: "ns-3"},
			},
			permissiveMode: false,

			sdsCertType:    envoy.RootCertTypeForMTLSInbound,
			requestedCerts: []string{"root-cert-for-mtls-inbound:ns-1/service-1"}, // root-cert requested

			// expectations
			expectedSANs:        []string{"sa-2.ns-2.cluster.local", "sa-3.ns-3.cluster.local"},
			expectedSecretCount: 1,
		},
		// Test case 1 end -------------------------------

		// Test case 2: root-cert-for-mtls-outbound requested -------------------------------
		{
			s: &sdsImpl{
				proxyServices: []service.MeshService{
					{Name: "service-1", Namespace: "ns-1"},
				},
				svcAccount:  service.K8sServiceAccount{Name: "sa-1", Namespace: "ns-1"},
				proxy:       envoy.NewProxy(certificate.CommonName(fmt.Sprintf("%s.%s.%s", uuid.New().String(), "sa-1", "ns-1")), nil),
				meshCatalog: mockCatalog,
				certManager: mockCertManager,
				cfg:         mockConfigurator,
			},
			certCN:       certificate.CommonName("sa-1.ns-1.cluster.local"),
			proxyService: service.MeshService{Name: "service-1", Namespace: "ns-1"},
			allowedDirectionalSvcAccounts: []service.K8sServiceAccount{
				{Name: "sa-2", Namespace: "ns-2"},
				{Name: "sa-3", Namespace: "ns-3"},
			},
			permissiveMode: false,

			sdsCertType:    envoy.RootCertTypeForMTLSOutbound,
			requestedCerts: []string{"root-cert-for-mtls-outbound:ns-1/service-1"}, // root-cert requested

			// expectations
			expectedSANs:        []string{"sa-2.ns-2.cluster.local", "sa-3.ns-3.cluster.local"},
			expectedSecretCount: 1,
		},
		// Test case 2 end -------------------------------

		// Test case 3: root-cert-for-https requested -------------------------------
		{
			s: &sdsImpl{
				proxyServices: []service.MeshService{
					{Name: "service-1", Namespace: "ns-1"},
				},
				svcAccount:  service.K8sServiceAccount{Name: "sa-1", Namespace: "ns-1"},
				proxy:       envoy.NewProxy(certificate.CommonName(fmt.Sprintf("%s.%s.%s", uuid.New().String(), "sa-1", "ns-1")), nil),
				meshCatalog: mockCatalog,
				certManager: mockCertManager,
				cfg:         mockConfigurator,
			},
			certCN:                        certificate.CommonName("sa-1.ns-1.cluster.local"),
			proxyService:                  service.MeshService{Name: "service-1", Namespace: "ns-1"},
			allowedDirectionalSvcAccounts: []service.K8sServiceAccount{},
			permissiveMode:                false,

			sdsCertType:    envoy.RootCertTypeForHTTPS,
			requestedCerts: []string{"root-cert-https:ns-1/service-1"}, // root-cert requested

			// expectations
			expectedSANs:        []string{},
			expectedSecretCount: 1,
		},
		// Test case 3 end -------------------------------

		// Test case 4: service-cert requested -------------------------------
		{
			s: &sdsImpl{
				proxyServices: []service.MeshService{
					{Name: "service-1", Namespace: "ns-1"},
				},
				svcAccount:  service.K8sServiceAccount{Name: "sa-1", Namespace: "ns-1"},
				proxy:       envoy.NewProxy(certificate.CommonName(fmt.Sprintf("%s.%s.%s", uuid.New().String(), "sa-1", "ns-1")), nil),
				meshCatalog: mockCatalog,
				certManager: mockCertManager,
				cfg:         mockConfigurator,
			},
			certCN:                        certificate.CommonName("sa-1.ns-1.cluster.local"),
			proxyService:                  service.MeshService{Name: "service-1", Namespace: "ns-1"},
			allowedDirectionalSvcAccounts: []service.K8sServiceAccount{},
			permissiveMode:                false,

			sdsCertType:    envoy.ServiceCertType,
			requestedCerts: []string{"service-cert:ns-1/service-1"}, // service-cert requested

			// expectations
			expectedSANs:        []string{},
			expectedSecretCount: 1,
		},
		// Test case 4 end -------------------------------

		// Test case 5: invalid cert type requested -------------------------------
		{
			s: &sdsImpl{
				proxyServices: []service.MeshService{
					{Name: "service-1", Namespace: "ns-1"},
				},
				svcAccount:  service.K8sServiceAccount{Name: "sa-1", Namespace: "ns-1"},
				proxy:       envoy.NewProxy(certificate.CommonName(fmt.Sprintf("%s.%s.%s", uuid.New().String(), "sa-1", "ns-1")), nil),
				meshCatalog: mockCatalog,
				certManager: mockCertManager,
				cfg:         mockConfigurator,
			},
			certCN:                        certificate.CommonName("sa-1.ns-1.cluster.local"),
			proxyService:                  service.MeshService{Name: "service-1", Namespace: "ns-1"},
			allowedDirectionalSvcAccounts: []service.K8sServiceAccount{},
			permissiveMode:                false,

			sdsCertType:    envoy.SDSCertType("invalid"),
			requestedCerts: []string{"invalid:ns-1/service-1"}, // service-cert requested

			// expectations
			expectedSANs:        []string{},
			expectedSecretCount: 0, // error is logged and no SDS secret is created
		},
		// Test case 5 end -------------------------------
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("Testing test case %d", i), func(t *testing.T) {
			// Mock calls based on test case input
			switch tc.sdsCertType {
			// Verify SAN for inbound and outbound MTLS certs
			case envoy.RootCertTypeForMTLSInbound:
				mockConfigurator.EXPECT().IsPermissiveTrafficPolicyMode().Return(tc.permissiveMode).Times(1)
				mockCatalog.EXPECT().ListAllowedInboundServiceAccounts(tc.s.svcAccount).Return(tc.allowedDirectionalSvcAccounts, nil).Times(1)
				mockCertificater.EXPECT().GetIssuingCA().Return([]byte("foo")).Times(1)

			case envoy.RootCertTypeForMTLSOutbound:
				mockConfigurator.EXPECT().IsPermissiveTrafficPolicyMode().Return(tc.permissiveMode).Times(1)
				mockCatalog.EXPECT().ListAllowedOutboundServiceAccounts(tc.s.svcAccount).Return(tc.allowedDirectionalSvcAccounts, nil).Times(1)
				mockCertificater.EXPECT().GetIssuingCA().Return([]byte("foo")).Times(1)

			case envoy.RootCertTypeForHTTPS:
				mockConfigurator.EXPECT().IsPermissiveTrafficPolicyMode().Return(tc.permissiveMode).Times(1)
				mockCertificater.EXPECT().GetIssuingCA().Return([]byte("foo")).Times(1)

			case envoy.ServiceCertType:
				mockCertificater.EXPECT().GetCertificateChain().Return([]byte("foo")).Times(1)
				mockCertificater.EXPECT().GetPrivateKey().Return([]byte("foo")).Times(1)
			}

			// test the function
			sdsSecrets := tc.s.getSDSSecrets(mockCertificater, tc.requestedCerts, tc.proxyService)
			assert.Len(sdsSecrets, tc.expectedSecretCount)

			if tc.expectedSecretCount <= 0 {
				// nothing to validate further
				return
			}

			sdsSecret := sdsSecrets[0]

			// verify the returned secret corresponds to the correct cert type
			assert.Equal(sdsSecret.Name, fmt.Sprintf("%s:%s", tc.sdsCertType, tc.proxyService))

			// verify different cert types
			switch tc.sdsCertType {
			// Verify SAN for inbound and outbound MTLS certs
			case envoy.RootCertTypeForMTLSInbound:
				fallthrough
			case envoy.RootCertTypeForMTLSOutbound:
				fallthrough
			case envoy.RootCertTypeForHTTPS:
				// build the list of SAN from the returned secret
				var actualSANs []string
				for _, stringMatcher := range sdsSecret.GetValidationContext().GetMatchSubjectAltNames() {
					actualSANs = append(actualSANs, stringMatcher.GetExact())
				}
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
