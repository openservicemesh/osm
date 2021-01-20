package tresor

import (
	"time"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/configurator"
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
		rootCertPem := "sample_certificate.pem"
		rootKeyPem := "sample_private_key.pem"
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
		rootCertPem := "sample_certificate.pem"
		rootKeyPem := "sample_private_key.pem"
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
})
