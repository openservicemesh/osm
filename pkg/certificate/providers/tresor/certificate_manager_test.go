package tresor

import (
	"time"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/configurator"
)

const (
	rootCertPem = "sample_certificate.pem"
	rootKeyPem  = "sample_private_key.pem"
)

var _ = Describe("Test Certificate Provider", func() {
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
		mockConfigurator.EXPECT().GetCertKeyBitSize().Return(2048).AnyTimes()

		rootCert, err := NewCA(cn, 1*time.Hour, rootCertCountry, rootCertLocality, rootCertOrganization)
		if err != nil {
			GinkgoT().Fatalf("Error loading CA from files %s and %s: %s", rootCertPem, rootKeyPem, err.Error())
		}
		m, newCertError := NewProvider(
			rootCert,
			"org",
			mockConfigurator.GetCertKeyBitSize(),
		)
		It("should issue a certificate", func() {
			Expect(newCertError).ToNot(HaveOccurred())
			cert, issueCertificateError := m.IssueCertificate(serviceFQDN, validity)
			Expect(issueCertificateError).ToNot(HaveOccurred())
			Expect(cert.GetCommonName()).To(Equal(certificate.CommonName(serviceFQDN)))

			x509Cert, err := certificate.DecodePEMCertificate(rootCert.GetCertificateChain())
			Expect(err).ToNot(HaveOccurred())

			issuingCAPEM, err := certificate.EncodeCertDERtoPEM(x509Cert.Raw)
			Expect(err).ToNot(HaveOccurred())
			Expect([]byte(cert.GetIssuingCA())).To(Equal([]byte(issuingCAPEM)))

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

	Context("Test nil certificate issue", func() {
		validity := 3 * time.Second
		mockConfigurator = configurator.NewMockConfigurator(mockCtrl)
		mockConfigurator.EXPECT().GetServiceCertValidityPeriod().Return(validity).AnyTimes()
		mockConfigurator.EXPECT().GetCertKeyBitSize().Return(2048).AnyTimes()
		m, newCertError := NewProvider(
			nil,
			"org",
			mockConfigurator.GetCertKeyBitSize(),
		)
		It("should return nil and error of no certificate", func() {
			Expect(m).To(BeNil())
			Expect(newCertError).To(Equal(errNoIssuingCA))
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
		mockConfigurator.EXPECT().GetCertKeyBitSize().Return(2048).AnyTimes()

		rootCert, err := NewCA(cn, validity, rootCertCountry, rootCertLocality, rootCertOrganization)
		if err != nil {
			GinkgoT().Fatalf("Error loading CA from files %s and %s: %s", rootCertPem, rootKeyPem, err.Error())
		}
		p, newCertError := NewProvider(
			rootCert,
			"org",
			mockConfigurator.GetCertKeyBitSize(),
		)
		It("should get an issued certificate from the cache", func() {
			Expect(newCertError).ToNot(HaveOccurred())
			cert, issueCertificateError := p.IssueCertificate(serviceFQDN, validity)
			Expect(issueCertificateError).ToNot(HaveOccurred())
			Expect(cert.GetCommonName()).To(Equal(certificate.CommonName(serviceFQDN)))

			manager := &certificate.CertManager{}
			cachedCert, getCertificateError := manager.GetCertificate(serviceFQDN)
			Expect(getCertificateError).ToNot(HaveOccurred())
			Expect(cachedCert).To(Equal(cert))
		})
	})

	Context("Test issuing a certificate when a root certificate is empty", func() {
		validity := 1 * time.Hour

		p := &Provider{}
		It("should return errNoIssuingCA error", func() {
			cert, issueCertificateError := p.IssueCertificate(serviceFQDN, validity)
			Expect(cert).To(BeNil())
			Expect(issueCertificateError).To(Equal(errNoIssuingCA))
		})
	})
})
