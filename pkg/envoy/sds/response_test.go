package sds

import (
	"fmt"
	"testing"

	xds_auth "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	xds_discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	xds_matcher "github.com/envoyproxy/go-control-plane/envoy/type/matcher/v3"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	tassert "github.com/stretchr/testify/assert"
	trequire "github.com/stretchr/testify/require"
	testclient "k8s.io/client-go/kubernetes/fake"

	"github.com/openservicemesh/osm/pkg/envoy/secrets"
	configFake "github.com/openservicemesh/osm/pkg/gen/client/config/clientset/versioned/fake"

	"github.com/openservicemesh/osm/pkg/catalog"
	catalogFake "github.com/openservicemesh/osm/pkg/catalog/fake"
	"github.com/openservicemesh/osm/pkg/certificate"
	tresorFake "github.com/openservicemesh/osm/pkg/certificate/providers/tresor/fake"
	"github.com/openservicemesh/osm/pkg/configurator"
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
	proxyIdentity := identity.K8sServiceAccount{Name: uuid.New().String(), Namespace: uuid.New().String()}.ToServiceIdentity()

	// This is the thing we are going to be requesting (pretending that the Envoy is requesting it)
	request := &xds_discovery.DiscoveryRequest{
		TypeUrl: string(envoy.TypeSDS),
		ResourceNames: []string{
			secrets.GetSDSServiceCertForIdentity(proxyIdentity).String(),
			secrets.GetSDSInboundRootCertForIdentity(proxyIdentity).String(),
		},
	}

	stop := make(chan struct{})
	defer close(stop)

	// The Common Name of the xDS Certificate (issued to the Envoy on the Pod by the Injector) will
	// have be prefixed with the ID of the pod. It is the first chunk of a dot-separated string.
	podID := uuid.New().String()

	certCommonName := certificate.CommonName(fmt.Sprintf("%s.%s.%s.", podID, envoy.KindSidecar, proxyIdentity))
	certSerialNumber := certificate.SerialNumber("123456")
	proxy, err := envoy.NewProxy(certCommonName, certSerialNumber, nil)
	assert.Nil(err)

	_, err = envoy.NewProxy("-certificate-common-name-is-invalid-", "-cert-serial-number-is-invalid-", nil)
	assert.Equal(err, envoy.ErrInvalidCertificateCN)

	cfg, err := configurator.NewConfigurator(fakeConfigClient, stop, "-osm-namespace-", "-the-mesh-config-name-", nil)
	assert.Nil(err)
	certManager := tresorFake.NewFake(nil)
	meshCatalog := catalogFake.NewFakeMeshCatalog(fakeKubeClient, fakeConfigClient)

	// ----- Test with an properly configured proxy
	resources, err := NewResponse(meshCatalog, proxy, request, cfg, certManager, nil)
	assert.Equal(err, nil, fmt.Sprintf("Error evaluating sds.NewResponse(): %s", err))
	assert.NotNil(resources)
	assert.Equal(len(resources), 2) // 1. service-cert, 2. root-cert-for-mtls-inbound (refer to the DiscoveryRequest 'request')
	_, ok := resources[0].(*xds_auth.Secret)
	assert.True(ok)
}

func TestGetSDSSecrets(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	// fakeCertManager, err := certificate.FakeCertManager()
	// if err != nil {
	// 	t.Error(err)
	// }

	cert := &certificate.Certificate{
		CertChain:  []byte("foo-chain"),
		PrivateKey: []byte("foo-key"),
		IssuingCA:  []byte("foo-ca"),
	}

	// This is used to dynamically set expectations for each test in the list of table driven tests
	type dynamicMock struct {
		mockCatalog      *catalog.MockMeshCataloger
		mockConfigurator *configurator.MockConfigurator
	}

	type testCase struct {
		name    string
		prepare func(d *dynamicMock)

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
			name: "test root-cert-for-mtls-inbound cert type request",

			prepare: func(d *dynamicMock) {},

			requestedCerts: []string{"root-cert-for-mtls-inbound:sa-1.ns-1.cluster.local"}, // root-cert requested

			// expectations
			expectedSANs:        nil,
			expectedSecretCount: 1,
		},
		// Test case 1 end -------------------------------

		// Test case 2: root-cert-for-mtls-outbound requested -------------------------------
		{
			name: "test root-cert-for-mtls-outbound cert type request",

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

			requestedCerts: []string{"root-cert-for-mtls-outbound:ns-2/service-2"}, // root-cert requested

			// expectations
			expectedSANs:        []string{"sa-2.ns-2.cluster.local", "sa-3.ns-2.cluster.local"},
			expectedSecretCount: 1,
		},
		// Test case 2 end -------------------------------

		// Test case 3: service-cert requested -------------------------------
		{
			name: "test service-cert cert type request",

			prepare: func(d *dynamicMock) {},

			requestedCerts: []string{"service-cert:sa-1.ns-1.cluster.local"}, // service-cert requested

			// expectations
			expectedSANs:        []string{},
			expectedSecretCount: 1,
		},
		// Test case 3 end -------------------------------

		// Test case 4: invalid cert type requested -------------------------------
		{
			name: "test invalid cert type request",

			prepare: nil,

			requestedCerts: []string{"invalid:ns-1/service-1"}, // service-cert requested

			// expectations
			expectedSANs:        []string{},
			expectedSecretCount: 0, // error is logged and no SDS secret is created
		},
		// Test case 4 end -------------------------------
		{
			name: "test service-cert invalid format",

			prepare: func(d *dynamicMock) {},

			requestedCerts: []string{"service-cert:ns-1/sa-1"}, // service-cert requested

			// expectations
			expectedSANs:        []string{},
			expectedSecretCount: 0,
		},
		{
			name: "test mtls inbound invalid format",

			prepare: func(d *dynamicMock) {},

			requestedCerts: []string{"root-cert-for-mtls-inbound:ns-1/sa-1"},

			// expectations
			expectedSANs:        []string{},
			expectedSecretCount: 0,
		},

		{
			name:           "test inbound MTLS certificate validation",
			requestedCerts: []string{"root-cert-for-mtls-inbound:sa-1.ns-1.cluster.local"},

			prepare: func(d *dynamicMock) {},

			expectedSecretCount: 1,
			// expectations
			expectedSANs: nil,
		},
		// Test case 1 end -------------------------------

		// Test case 2: tests SDS secret for outbound TLS secret -------------------------------
		{
			name:           "test outbound MTLS certificate validation with 2 expected SANs",
			requestedCerts: []string{"root-cert-for-mtls-outbound:ns-2/sa-2"},
			prepare: func(d *dynamicMock) {
				associatedSvcAccounts := []identity.ServiceIdentity{
					identity.K8sServiceAccount{Name: "sa-2", Namespace: "ns-2"}.ToServiceIdentity(),
					identity.K8sServiceAccount{Name: "sa-3", Namespace: "ns-2"}.ToServiceIdentity(),
				}
				d.mockCatalog.EXPECT().ListServiceIdentitiesForService(service.MeshService{
					Name:      "sa-2",
					Namespace: "ns-2",
				}).Return(associatedSvcAccounts).Times(1)
			},

			expectedSecretCount: 1,
			// expectations
			expectedSANs: []string{"sa-2.ns-2.cluster.local", "sa-3.ns-2.cluster.local"},
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("Testing test case %d: %s", i, tc.name), func(t *testing.T) {
			assert := tassert.New(t)
			require := trequire.New(t)
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			// Initialize the dynamic mocks
			d := dynamicMock{
				mockCatalog:      catalog.NewMockMeshCataloger(mockCtrl),
				mockConfigurator: configurator.NewMockConfigurator(mockCtrl),
			}

			// Prepare the dynamic mock expectations for each test case
			if tc.prepare != nil {
				tc.prepare(&d)
			}

			certCommonName := certificate.CommonName(fmt.Sprintf("%s.%s.%s.%s", uuid.New(), envoy.KindSidecar, "sa-1", "ns-1"))
			certSerialNumber := certificate.SerialNumber("123456")

			proxy, err := envoy.NewProxy(certCommonName, certSerialNumber, nil)
			assert.Nil(err)

			// test the function
			sdsSecrets := getSDSSecrets(d.mockCatalog, cert, tc.requestedCerts, proxy)

			require.Len(sdsSecrets, tc.expectedSecretCount)

			if tc.expectedSecretCount <= 0 {
				return
			}

			sdsSecret := sdsSecrets[0]

			for _, requested := range tc.requestedCerts {
				sdsCert, err := secrets.UnmarshalSDSCert(requested)
				// if we passed tc.expectedSecretCoutn >= 0 then this should not be an error.
				assert.Nil(err)

				switch sdsCert.(type) {
				// Verify SAN for inbound and outbound MTLS certs
				case *secrets.SDSInboundRootCert, *secrets.SDSOutboundRootCert:
					// Check SANs
					actualSANs := subjectAltNamesToStr(sdsSecret.GetValidationContext().GetMatchSubjectAltNames())
					assert.ElementsMatch(actualSANs, tc.expectedSANs)

					// Check trusted CA
					assert.NotNil(sdsSecret.GetValidationContext().GetTrustedCa().GetInlineBytes())
				case *secrets.SDSServiceCert:
					assert.NotNil(sdsSecret.GetTlsCertificate().GetCertificateChain().GetInlineBytes())
					assert.NotNil(sdsSecret.GetTlsCertificate().GetPrivateKey().GetInlineBytes())

					assert.NotNil(sdsSecret)
					assert.Equal((sdsSecret.GetTlsCertificate().GetCertificateChain().GetInlineBytes()), []byte(cert.GetCertificateChain()))
					assert.Equal(sdsSecret.GetTlsCertificate().GetPrivateKey().GetInlineBytes(), []byte(cert.GetPrivateKey()))
				}
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

			actual := getSubjectAltNamesFromSvcIdentities(tc.serviceIdentities)
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
