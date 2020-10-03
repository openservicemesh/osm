package tresor

import (
	"time"

	"github.com/openservicemesh/osm/pkg/certificate"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Test Certificate Manager", func() {
	defer GinkgoRecover()
	const (
		serviceFQDN = "a.b.c"
		rootFQDN    = "bookbuyer.azure.mesh"
	)

	Context("Test issuing a certificate from a CA loaded from file", func() {
		validity := 3 * time.Second
		rootCertPem := "../../sample_certificate.pem"
		rootKeyPem := "../../sample_private_key.pem"

		It("should be able to load a certificate from disk", func() {
			var err error
			rootCert, err := LoadCA(rootCertPem, rootKeyPem)
			Expect(err).ToNot(HaveOccurred())

			expected := "-----BEGIN CERTIFICATE-----\nMIIElzCCA3+gAwIBAgIRAOsakgIV4y"
			Expect(string(rootCert.GetCertificateChain()[:len(expected)])).To(Equal(expected))

			m, newCertError := NewCertManager(rootCert, validity, "org")
			Expect(newCertError).ToNot(HaveOccurred())

			cert, err := m.IssueCertificate(serviceFQDN, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(cert.GetCommonName()).To(Equal(certificate.CommonName(serviceFQDN)))

			x509RootCert, err := certificate.DecodePEMCertificate(rootCert.GetCertificateChain())
			Expect(err).ToNot(HaveOccurred())

			issuingCAPEM, err := certificate.EncodeCertDERtoPEM(x509RootCert.Raw)
			Expect(err).ToNot(HaveOccurred())
			Expect(cert.GetIssuingCA()).To(Equal([]byte(issuingCAPEM)))

			pemCert := cert.GetCertificateChain()
			x509Cert, err := certificate.DecodePEMCertificate(pemCert)
			Expect(err).ToNot(HaveOccurred())
			Expect(x509Cert.Subject.CommonName).To(Equal(serviceFQDN))

			pemRootCert := cert.GetIssuingCA()
			xRootCert, err := certificate.DecodePEMCertificate(pemRootCert)
			Expect(err).ToNot(HaveOccurred())
			Expect(xRootCert.Subject.CommonName).To(Equal(rootFQDN))
		})
	})

	Context("Test issuing a certificate from a newly created CA", func() {
		validity := 3 * time.Second
		rootCertPem := "sample_certificate.pem"
		rootKeyPem := "sample_private_key.pem"
		cn := certificate.CommonName("Test CA")
		rootCertCountry := "US"
		rootCertLocality := "CA"
		rootCertOrganization := "Open Service Mesh Tresor"
		rootCert, err := NewCA(cn, 1*time.Hour, rootCertCountry, rootCertLocality, rootCertOrganization)
		if err != nil {
			GinkgoT().Fatalf("Error loading CA from files %s and %s: %s", rootCertPem, rootKeyPem, err.Error())
		}
		m, newCertError := NewCertManager(rootCert, validity, "org")
		It("should issue a certificate", func() {
			Expect(newCertError).ToNot(HaveOccurred())
			cert, issueCertificateError := m.IssueCertificate(serviceFQDN, nil)
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
		rootCert, err := NewCA(cn, validity, rootCertCountry, rootCertLocality, rootCertOrganization)
		if err != nil {
			GinkgoT().Fatalf("Error loading CA from files %s and %s: %s", rootCertPem, rootKeyPem, err.Error())
		}
		m, newCertError := NewCertManager(rootCert, validity, "org")
		It("should get an issued certificate from the cache", func() {
			Expect(newCertError).ToNot(HaveOccurred())
			cert, issueCertificateError := m.IssueCertificate(serviceFQDN, &validity)
			Expect(issueCertificateError).ToNot(HaveOccurred())
			Expect(cert.GetCommonName()).To(Equal(certificate.CommonName(serviceFQDN)))

			cachedCert, getCertificateError := m.GetCertificate(serviceFQDN)
			Expect(getCertificateError).ToNot(HaveOccurred())
			Expect(cachedCert).To(Equal(cert))
		})
	})
})
