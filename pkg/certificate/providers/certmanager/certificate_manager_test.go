package certmanager

import (
	"crypto/rand"
	"crypto/x509"
	"testing"
	"time"

	tassert "github.com/stretchr/testify/assert"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/golang/mock/gomock"
	cmapi "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1"
	cmmeta "github.com/jetstack/cert-manager/pkg/apis/meta/v1"
	cmfakeclient "github.com/jetstack/cert-manager/pkg/client/clientset/versioned/fake"
	cmfakeapi "github.com/jetstack/cert-manager/pkg/client/clientset/versioned/typed/certmanager/v1/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	test "k8s.io/client-go/testing"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/messaging"
	"github.com/openservicemesh/osm/pkg/tests"
)

var (
	mockCtrl         = gomock.NewController(GinkgoT())
	mockConfigurator = configurator.NewMockConfigurator(mockCtrl)
	cn               = certificate.CommonName("bookbuyer.azure.mesh")
	crNotReady       = &cmapi.CertificateRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "osm-123",
			Namespace: "osm-system",
		},
	}
)

const (
	validity = 1 * time.Hour
	keySize  = 2048
)

var _ = Describe("Test cert-manager Certificate Manager", func() {
	defer GinkgoRecover()

	Context("Test Getting a certificate from the cache", func() {
		rootCertPEM, err := tests.GetPEMCert()
		if err != nil {
			GinkgoT().Fatalf("Error loading sample test certificate: %s", err.Error())
		}

		rootCert, err := certificate.DecodePEMCertificate(rootCertPEM)
		if err != nil {
			GinkgoT().Fatalf("Error decoding certificate from file: %s", err.Error())
		}
		Expect(rootCert).ToNot(BeNil())
		rootCert.NotAfter = time.Now().Add(time.Minute * 30)

		rootKeyPEM, err := tests.GetPEMPrivateKey()
		if err != nil {
			GinkgoT().Fatalf("Error loading private key: %s", err.Error())
		}
		rootKey, err := certificate.DecodePEMPrivateKey(rootKeyPEM)
		if err != nil {
			GinkgoT().Fatalf("Error decoding private key: %s", err.Error())
		}
		Expect(rootKey).ToNot(BeNil())

		signedCertDER, err := x509.CreateCertificate(rand.Reader, rootCert, rootCert, rootKey.Public(), rootKey)
		if err != nil {
			GinkgoT().Fatalf("Failed to self signed certificate: %s", err.Error())
		}

		signedCertPEM, err := certificate.EncodeCertDERtoPEM(signedCertDER)
		if err != nil {
			GinkgoT().Fatalf("Failed encode signed signed certificate: %s", err.Error())
		}

		rootCertificator, err := NewRootCertificateFromPEM(rootCertPEM)
		if err != nil {
			GinkgoT().Fatalf("Error loading ca %s: %s", rootCertPEM, err.Error())
		}

		crReady := crNotReady.DeepCopy()
		crReady.Status = cmapi.CertificateRequestStatus{
			Certificate: signedCertPEM,
			CA:          signedCertPEM,
			Conditions: []cmapi.CertificateRequestCondition{
				{
					Type:   cmapi.CertificateRequestConditionReady,
					Status: cmmeta.ConditionTrue,
				},
			},
		}

		fakeClient := cmfakeclient.NewSimpleClientset()
		fakeClient.CertmanagerV1().(*cmfakeapi.FakeCertmanagerV1).Fake.PrependReactor("*", "*", func(action test.Action) (bool, runtime.Object, error) {
			switch action.GetVerb() {
			case "create":
				return true, crNotReady, nil
			case "get":
				return true, crReady, nil
			case "list":
				return true, &cmapi.CertificateRequestList{Items: []cmapi.CertificateRequest{*crReady}}, nil
			case "delete":
				return true, nil, nil
			default:
				return false, nil, nil
			}
		})

		stop := make(chan struct{})
		defer close(stop)
		msgBroker := messaging.NewBroker(stop)

		cm, newCertError := NewCertManager(
			rootCertificator,
			fakeClient,
			"osm-system",
			cmmeta.ObjectReference{Name: "osm-ca"},
			mockConfigurator,
			mockConfigurator.GetServiceCertValidityPeriod(),
			mockConfigurator.GetCertKeyBitSize(),
			msgBroker,
		)
		It("should get an issued certificate from the cache", func() {
			mockConfigurator.EXPECT().GetCertKeyBitSize().Return(keySize).AnyTimes()

			Expect(newCertError).ToNot(HaveOccurred())
			cert, issueCertificateError := cm.IssueCertificate(cn, validity)
			Expect(issueCertificateError).ToNot(HaveOccurred())
			Expect(cert.GetCommonName()).To(Equal(cn))

			cachedCert, getCertificateError := cm.GetCertificate(cn)
			Expect(getCertificateError).ToNot(HaveOccurred())
			Expect(cachedCert).To(Equal(cert))
		})

		It("should rotate the certificate", func() {
			mockConfigurator.EXPECT().GetServiceCertValidityPeriod().Return(validity).AnyTimes()
			mockConfigurator.EXPECT().GetCertKeyBitSize().Return(keySize).AnyTimes()

			cert, err := cm.RotateCertificate(cn)
			Expect(err).Should(BeNil())
			cachedCert, err := cm.GetCertificate(cn)
			Expect(cachedCert).To(Equal(cert))
			Expect(err).Should(BeNil())
		})
	})
})

func TestReleaseCertificate(t *testing.T) {
	cert := &Certificate{
		commonName: cn,
		expiration: time.Now().Add(1 * time.Hour),
	}
	manager := &CertManager{cache: map[certificate.CommonName]certificate.Certificater{cn: cert}}

	testCases := []struct {
		name       string
		commonName certificate.CommonName
	}{
		{
			name:       "release existing certificate",
			commonName: cn,
		},
		{
			name:       "release non-existing certificate",
			commonName: cn,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := tassert.New(t)

			manager.ReleaseCertificate(tc.commonName)
			_, err := manager.GetCertificate(tc.commonName)
			assert.ErrorIs(err, errCertNotFound)
		})
	}
}

func TestGetRootCertificate(t *testing.T) {
	assert := tassert.New(t)

	manager := &CertManager{
		ca: &Certificate{
			commonName: cn,
			expiration: time.Now().Add(1 * time.Hour),
		},
	}
	cert, err := manager.GetRootCertificate()

	assert.Nil(err)
	assert.Equal(manager.ca, cert)
}

func TestCertificaterFromCertificateRequest(t *testing.T) {
	assert := tassert.New(t)
	fakeClient := cmfakeclient.NewSimpleClientset()

	rootCertPEM, err := tests.GetPEMCert()
	assert.Nil(err)

	rootCert, err := certificate.DecodePEMCertificate(rootCertPEM)
	assert.Nil(err)

	rootKeyPEM, err := tests.GetPEMPrivateKey()
	assert.Nil(err)

	rootKey, err := certificate.DecodePEMPrivateKey(rootKeyPEM)
	assert.Nil(err)

	rootCertificator, err := NewRootCertificateFromPEM(rootCertPEM)
	assert.Nil(err)

	mockConfigurator.EXPECT().GetServiceCertValidityPeriod().Return(validity).AnyTimes()
	mockConfigurator.EXPECT().GetCertKeyBitSize().Return(keySize).AnyTimes()

	cm, err := NewCertManager(
		rootCertificator,
		fakeClient,
		"osm-system",
		cmmeta.ObjectReference{Name: "osm-ca"},
		mockConfigurator,
		mockConfigurator.GetServiceCertValidityPeriod(),
		mockConfigurator.GetCertKeyBitSize(),
		nil,
	)
	assert.Nil(err)

	signedCertDER, err := x509.CreateCertificate(rand.Reader, rootCert, rootCert, rootKey.Public(), rootKey)
	assert.Nil(err)

	signedCertPEM, err := certificate.EncodeCertDERtoPEM(signedCertDER)
	assert.Nil(err)

	crReady := crNotReady.DeepCopy()
	crReady.Status = cmapi.CertificateRequestStatus{
		Certificate: signedCertPEM,
		CA:          signedCertPEM,
		Conditions: []cmapi.CertificateRequestCondition{
			{
				Type:   cmapi.CertificateRequestConditionReady,
				Status: cmmeta.ConditionTrue,
			},
		},
	}
	emptyArr := []byte{}
	testCases := []struct {
		name              string
		cr                cmapi.CertificateRequest
		expectedCertIsNil bool
		expectedError     error
	}{
		{
			name:              "Could not decode PEM Cert",
			cr:                *crNotReady,
			expectedCertIsNil: true,
			expectedError:     certificate.ErrNoCertificateInPEM,
		},
		{
			name:              "default",
			cr:                *crReady,
			expectedCertIsNil: false,
			expectedError:     nil,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := tassert.New(t)

			cert, err := cm.certificaterFromCertificateRequest(&tc.cr, emptyArr)

			assert.Equal(tc.expectedCertIsNil, cert == nil)
			assert.Equal(tc.expectedError, err)
		})
	}
	// Tests if cmapi.CertificateRequest is nil
	cert, err := cm.certificaterFromCertificateRequest(nil, emptyArr)
	assert.Nil(cert)
	assert.Nil(err)
}
