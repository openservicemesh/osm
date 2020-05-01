package tresor

import (
	"time"

	"github.com/open-service-mesh/osm/pkg/certificate"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Test Certificate Manager", func() {

	const (
		serviceFQDN = "a.b.c"
		rootFQDN    = "bookbuyer.azure.mesh"
	)

	Context("Test issuing a certificate from a CA loaded from file", func() {
		validity := 3 * time.Second
		rootCertPem := "sample_certificate.pem"
		rootKeyPem := "sample_private_key.pem"

		It("should be able to load a certificate from disk", func() {
			var err error
			rootCert, err := LoadCA(rootCertPem, rootKeyPem)
			Expect(err).ToNot(HaveOccurred())

			expected := "-----BEGIN CERTIFICATE-----\nMIIElzCCA3+gAwIBAgIRAOsakgIV4y"
			Expect(string(rootCert.GetCertificateChain()[:len(expected)])).To(Equal(expected))

			m, newCertError := NewCertManager(rootCert, validity)
			Expect(newCertError).ToNot(HaveOccurred())

			cert, err := m.IssueCertificate(serviceFQDN)
			Expect(err).ToNot(HaveOccurred())
			Expect(cert.GetName()).To(Equal(serviceFQDN))

			x509RootCert, err := DecodePEMCertificate(rootCert.GetCertificateChain())
			Expect(err).ToNot(HaveOccurred())

			issuingCAPEM, err := encodeCertDERtoPEM(x509RootCert.Raw)
			Expect(err).ToNot(HaveOccurred())
			Expect(cert.GetIssuingCA()).To(Equal([]byte(issuingCAPEM)))

			pemCert := cert.GetCertificateChain()
			x509Cert, err := DecodePEMCertificate(pemCert)
			Expect(err).ToNot(HaveOccurred())
			Expect(x509Cert.Subject.CommonName).To(Equal(serviceFQDN))

			pemRootCert := cert.GetIssuingCA()
			xRootCert, err := DecodePEMCertificate(pemRootCert)
			Expect(err).ToNot(HaveOccurred())
			Expect(xRootCert.Subject.CommonName).To(Equal(rootFQDN))
		})
	})

	Context("Test issuing a certificate from a newly created CA", func() {
		validity := 3 * time.Second
		rootCertPem := "sample_certificate.pem"
		rootKeyPem := "sample_private_key.pem"
		cn := certificate.CommonName("Test CA")
		rootCert, err := NewCA(cn, 1*time.Hour)
		if err != nil {
			log.Fatal().Err(err).Msgf("Error loading CA from files %s and %s", rootCertPem, rootKeyPem)
		}
		m, newCertError := NewCertManager(rootCert, validity)
		It("should issue a certificate", func() {
			Expect(newCertError).ToNot(HaveOccurred())
			cert, issueCerterror := m.IssueCertificate(serviceFQDN)
			Expect(issueCerterror).ToNot(HaveOccurred())
			Expect(cert.GetName()).To(Equal(serviceFQDN))

			x509Cert, err := DecodePEMCertificate(rootCert.GetCertificateChain())
			Expect(err).ToNot(HaveOccurred())

			issuingCAPEM, err := encodeCertDERtoPEM(x509Cert.Raw)
			Expect(err).ToNot(HaveOccurred())
			Expect(cert.GetIssuingCA()).To(Equal([]byte(issuingCAPEM)))

			pemCert := cert.GetCertificateChain()
			xCert, err := DecodePEMCertificate(pemCert)
			Expect(err).ToNot(HaveOccurred())
			Expect(xCert.Subject.CommonName).To(Equal(serviceFQDN))

			pemRootCert := cert.GetIssuingCA()
			xRootCert, err := DecodePEMCertificate(pemRootCert)
			Expect(err).ToNot(HaveOccurred(), string(pemRootCert))
			Expect(xRootCert.Subject.CommonName).To(Equal(cn.String()))
		})
	})
})
