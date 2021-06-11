package tresor

import (
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	tassert "github.com/stretchr/testify/assert"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/configurator"
)

const (
	rootCertPem = "sample_certificate.pem"
	rootKeyPem  = "sample_private_key.pem"
)

var _ = Describe("Test Certificate Manager", func() {
	defer GinkgoRecover()

	var (
		mockCtrl         *gomock.Controller
		mockConfigurator *configurator.MockConfigurator
	)
	mockCtrl = gomock.NewController(GinkgoT())

	const serviceFQDN = "a.b.c"

	Context("Test issuing a certificate from a newly created CA", func() {
		validity := 3 * time.Second
		cn := certificate.CommonName("Test CA")
		rootCertCountry := "US"
		rootCertLocality := "CA"
		rootCertOrganization := "Open Service Mesh Tresor"

		mockConfigurator = configurator.NewMockConfigurator(mockCtrl)
		mockConfigurator.EXPECT().GetServiceCertValidityPeriod().Return(validity).AnyTimes()

		rootCert, err := NewCA(cn, 1*time.Hour, rootCertCountry, rootCertLocality, rootCertOrganization)
		if err != nil {
			GinkgoT().Fatalf("Error loading CA from files %s and %s: %s", rootCertPem, rootKeyPem, err.Error())
		}
		m, newCertError := NewCertManager(rootCert, "org", mockConfigurator)
		It("should issue a certificate", func() {
			Expect(newCertError).ToNot(HaveOccurred())
			cert, issueCertificateError := m.IssueCertificate(serviceFQDN, validity)
			Expect(issueCertificateError).ToNot(HaveOccurred())
			Expect(cert.GetCommonName()).To(Equal(certificate.CommonName(serviceFQDN)))

			x509Cert, err := certificate.DecodePEMCertificate(rootCert.GetCertificateChain())
			Expect(err).ToNot(HaveOccurred())

			issuingCAPEM, err := certificate.EncodeCertDERtoPEM(x509Cert.Raw)
			Expect(err).ToNot(HaveOccurred())
			Expect(cert.GetIssuingCA()).To(Equal([]byte(issuingCAPEM)))

			pemCert := cert.GetCertificateChain()
			xCert, err := certificate.DecodePEMCertificate(pemCert)
			Expect(err).ToNot(HaveOccurred())
			Expect(xCert.Subject.CommonName).To(Equal(serviceFQDN))

			pemRootCert := cert.GetIssuingCA()
			xRootCert, err := certificate.DecodePEMCertificate(pemRootCert)
			Expect(err).ToNot(HaveOccurred(), string(pemRootCert))
			Expect(xRootCert.Subject.CommonName).To(Equal(cn.String()))
		})
	})

	Context("Test Getting a certificate from the cache", func() {
		validity := 1 * time.Hour
		cn := certificate.CommonName("Test CA")
		rootCertCountry := "US"
		rootCertLocality := "CA"
		rootCertOrganization := "Open Service Mesh Tresor"

		mockConfigurator = configurator.NewMockConfigurator(mockCtrl)
		mockConfigurator.EXPECT().GetServiceCertValidityPeriod().Return(validity).AnyTimes()

		rootCert, err := NewCA(cn, validity, rootCertCountry, rootCertLocality, rootCertOrganization)
		if err != nil {
			GinkgoT().Fatalf("Error loading CA from files %s and %s: %s", rootCertPem, rootKeyPem, err.Error())
		}
		m, newCertError := NewCertManager(rootCert, "org", mockConfigurator)
		It("should get an issued certificate from the cache", func() {
			Expect(newCertError).ToNot(HaveOccurred())
			cert, issueCertificateError := m.IssueCertificate(serviceFQDN, validity)
			Expect(issueCertificateError).ToNot(HaveOccurred())
			Expect(cert.GetCommonName()).To(Equal(certificate.CommonName(serviceFQDN)))

			cachedCert, getCertificateError := m.GetCertificate(serviceFQDN)
			Expect(getCertificateError).ToNot(HaveOccurred())
			Expect(cachedCert).To(Equal(cert))
		})
	})

	Context("Test issuing a certificate when a root certificate is empty", func() {
		validity := 1 * time.Hour

		m := &CertManager{}
		It("should return errNoIssuingCA error", func() {
			cert, issueCertificateError := m.IssueCertificate(serviceFQDN, validity)
			Expect(cert).To(BeNil())
			Expect(issueCertificateError).To(Equal(errNoIssuingCA))
		})
	})
})

func TestReleaseCertificate(t *testing.T) {
	cn := certificate.CommonName("Test CN")
	cert := &Certificate{
		commonName: cn,
		expiration: time.Now().Add(1 * time.Hour),
	}

	manager := &CertManager{}
	manager.cache.Store(cn, cert)

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

func TestGetCertificate(t *testing.T) {
	cn := certificate.CommonName("Test Cert")
	cert := &Certificate{
		commonName: cn,
		expiration: time.Now().Add(1 * time.Hour),
	}

	expiredCn := certificate.CommonName("Expired Test Cert")
	expiredCert := &Certificate{
		commonName: expiredCn,
		expiration: time.Now().Add(-1 * time.Hour),
	}

	manager := &CertManager{}
	manager.cache.Store(cn, cert)
	manager.cache.Store(expiredCn, expiredCert)

	testCases := []struct {
		name                string
		commonName          certificate.CommonName
		expectedCertificate *Certificate
		expectedErr         error
	}{
		{
			name:                "cache hit",
			commonName:          cn,
			expectedCertificate: cert,
		},
		{
			name:        "cache miss",
			commonName:  certificate.CommonName("Wrong Cert"),
			expectedErr: errCertNotFound,
		},
		{
			name:        "certificate expiration",
			commonName:  expiredCn,
			expectedErr: errCertNotFound,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := tassert.New(t)

			c, err := manager.GetCertificate(tc.commonName)
			if tc.expectedErr != nil {
				assert.ErrorIs(err, tc.expectedErr)
				return
			}

			assert.Nil(err)
			assert.Equal(tc.expectedCertificate, c)
		})
	}
}

func TestRotateCertificate(t *testing.T) {
	validity := 1 * time.Hour

	ca := certificate.CommonName("Test CA")
	rootCertCountry := "US"
	rootCertLocality := "CA"
	rootCertOrganization := "Open Service Mesh"

	rootCert, err := NewCA(ca, validity, rootCertCountry, rootCertLocality, rootCertOrganization)
	if err != nil {
		t.Fatalf("Error loading CA from files %s and %s: %s", rootCertPem, rootKeyPem, err)
	}

	cn := certificate.CommonName("Test Cert")

	oldCert := &Certificate{
		commonName: cn,
		expiration: time.Now().Add(-1 * time.Hour),
	}

	mockCtrl := gomock.NewController(t)
	mockConfigurator := configurator.NewMockConfigurator(mockCtrl)
	mockConfigurator.EXPECT().GetServiceCertValidityPeriod().Return(validity).AnyTimes()

	manager := &CertManager{ca: rootCert, cfg: mockConfigurator}
	manager.cache.Store(cn, oldCert)

	testCases := []struct {
		name           string
		commonName     certificate.CommonName
		expectedErrMsg string
	}{
		{
			name:           "non-existing cert",
			commonName:     certificate.CommonName("Wrong Cert"),
			expectedErrMsg: "Old certificate does not exist for CN=Wrong Cert",
		},
		{
			name:       "existing cert",
			commonName: cn,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := tassert.New(t)

			newCert, err := manager.RotateCertificate(tc.commonName)
			if len(tc.expectedErrMsg) != 0 {
				assert.EqualError(err, tc.expectedErrMsg)
				return
			}

			assert.Nil(err)
			assert.Equal(tc.commonName, newCert.GetCommonName())
			assert.True(newCert.GetExpiration().After(oldCert.GetExpiration()))

			cert, err := manager.GetCertificate(tc.commonName)
			assert.Nil(err)
			assert.Equal(newCert, cert)
		})
	}
}

func TestListCertificate(t *testing.T) {
	assert := tassert.New(t)

	cn := certificate.CommonName("Test Cert")
	cert := &Certificate{
		commonName: cn,
	}

	anotherCn := certificate.CommonName("Another Test Cert")
	anotherCert := &Certificate{
		commonName: anotherCn,
	}

	expectedCertificates := []*Certificate{cert, anotherCert}

	manager := &CertManager{}
	manager.cache.Store(cn, cert)
	manager.cache.Store(anotherCn, anotherCert)

	cs, err := manager.ListCertificates()

	assert.Nil(err)
	assert.Len(cs, 2)

	for i, c := range cs {
		match := false
		for _, ec := range expectedCertificates {
			if c.GetCommonName() == ec.GetCommonName() {
				match = true
				assert.Equal(ec, c)
				break
			}
		}

		if !match {
			t.Fatalf("Certificate #%v %v does not exist", i, c.GetCommonName())
		}
	}
}

func TestGetRootCertificate(t *testing.T) {
	assert := tassert.New(t)

	validity := 1 * time.Hour

	ca := certificate.CommonName("Test CA")
	rootCertCountry := "US"
	rootCertLocality := "CA"
	rootCertOrganization := "Open Service Mesh"

	rootCert, err := NewCA(ca, validity, rootCertCountry, rootCertLocality, rootCertOrganization)
	if err != nil {
		t.Fatalf("Error loading CA from files %s and %s: %s", rootCertPem, rootKeyPem, err)
	}

	manager := &CertManager{ca: rootCert}

	got, err := manager.GetRootCertificate()

	assert.Nil(err)
	assert.Equal(rootCert, got)
}
