package tresor

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/openservicemesh/osm/pkg/certificate"
)

var _ = Describe("Test creation of a new CA", func() {
	Context("Create a new CA", func() {
		rootCertCountry := "US"
		rootCertLocality := "CA"
		cert, err := NewCA("Tresor CA for Testing", 2*time.Second, rootCertCountry, rootCertLocality, rootCertOrganization)
		It("should create a new CA", func() {
			Expect(err).ToNot(HaveOccurred())

			x509Cert, err := certificate.DecodePEMCertificate(cert.GetCertificateChain())
			Expect(err).ToNot(HaveOccurred())

			Expect(x509Cert.NotAfter.Sub(x509Cert.NotBefore)).To(Equal(2 * time.Second))
			Expect(x509Cert.KeyUsage).To(Equal(x509.KeyUsageCertSign | x509.KeyUsageCRLSign))
			Expect(x509Cert.IsCA).To(BeTrue())
		})
	})
})

var _ = Describe("Test creation from pem", func() {
	Context("valid pem cert and pem key", func() {
		cn := certificate.CommonName("Test CA")
		rootCertCountry := "US"
		rootCertLocality := "CA"
		rootCertOrganization := "Root Cert Organization"

		notBefore := time.Now()
		notAfter := notBefore.Add(1 * time.Hour)
		serialNumber := big.NewInt(1)

		template := &x509.Certificate{
			SerialNumber: serialNumber,
			Subject: pkix.Name{
				CommonName:   cn.String(),
				Country:      []string{rootCertCountry},
				Locality:     []string{rootCertLocality},
				Organization: []string{rootCertOrganization},
			},
			NotBefore:             notBefore,
			NotAfter:              notAfter,
			KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
			BasicConstraintsValid: true,
			IsCA:                  true,
		}

		rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
		Expect(err).ToNot(HaveOccurred())

		derBytes, err := x509.CreateCertificate(rand.Reader, template, template, &rsaKey.PublicKey, rsaKey)
		Expect(err).ToNot(HaveOccurred())

		pemCert, err := certificate.EncodeCertDERtoPEM(derBytes)
		Expect(err).ToNot(HaveOccurred())

		pemKey, err := certificate.EncodeKeyDERtoPEM(rsaKey)
		Expect(err).ToNot(HaveOccurred())

		expiration := time.Now().Add(1 * time.Hour)

		c, err := NewCertificateFromPEM(pemCert, pemKey, expiration)
		Expect(err).ToNot(HaveOccurred())

		It("Should have the correct CN", func() {
			Expect(c.GetCommonName()).To(Equal(cn))
		})

		It("should decode PEM to x509 ", func() {
			Expect(err).ToNot(HaveOccurred())

			x509Cert, err := certificate.DecodePEMCertificate(c.GetCertificateChain())
			Expect(err).ToNot(HaveOccurred())

			Expect(x509Cert.NotAfter.Sub(x509Cert.NotBefore)).To(Equal(1 * time.Hour))
			Expect(x509Cert.KeyUsage).To(Equal(x509.KeyUsageCertSign | x509.KeyUsageCRLSign))
			Expect(x509Cert.IsCA).To(BeTrue())
			Expect(x509Cert.Subject.CommonName).To(Equal(cn.String()))
		})
	})

	Context("Test NewCertificateFromPEM()", func() {
		expiration := time.Now().Add(1 * time.Hour)

		It("invalid pem cert and pem key should returns en error", func() {
			_, err := NewCertificateFromPEM([]byte(""), []byte(""), expiration)
			Expect(err).To(HaveOccurred())
		})
	})
})
