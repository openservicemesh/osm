package certificate

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/openservicemesh/osm/pkg/certificate/pem"
	"github.com/openservicemesh/osm/pkg/tests/certificates"
)

var _ = Describe("Test Tresor Tools", func() {
	Context("Test EncodeCertDERtoPEM function", func() {
		cert, err := EncodeCertDERtoPEM([]byte{1, 2, 3})
		It("should have encoded DER bytes into a PEM certificate", func() {
			Expect(err).ShouldNot(HaveOccurred())
			Expect(cert).NotTo(Equal(nil))
		})
	})

	Context("Test EncodeCertReqDERtoPEM function", func() {
		cert, err := EncodeCertReqDERtoPEM([]byte{1, 2, 3})
		It("should have encoded DER bytes into a PEM certificate request", func() {
			Expect(err).ShouldNot(HaveOccurred())
			Expect(cert).NotTo(Equal(nil))
		})
	})

	Context("Test EncodeKeyDERtoPEM function", func() {
		pemKey := pem.PrivateKey(certificates.SamplePrivateKeyPEM)
		privKey, err := DecodePEMPrivateKey(pemKey)
		It("decodes PEM key to RSA Private Key", func() {
			Expect(err).ToNot(HaveOccurred(), string(pemKey))
		})

		It("loaded PEM key from file", func() {
			actual, err := EncodeKeyDERtoPEM(privKey)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(string(actual)).To(Equal(certificates.SamplePrivateKeyPEM))

			expected := "-----BEGIN PRIVATE KEY-----\nMIIEvQIBADANBgkqhkiG9w0BAQ"
			Expect(string(pemKey[:len(expected)])).To(Equal(expected))
		})
	})
})

var _ = Describe("Test tools", func() {
	Context("Testing decoding of PEMs", func() {
		It("should have decoded the PEM into x509 certificate", func() {
			x509Cert, err := DecodePEMCertificate([]byte(certificates.SampleCertificatePEM))
			Expect(err).ToNot(HaveOccurred())
			Expect(x509Cert.Subject.CommonName).To(Equal("bookbuyer.azure.mesh"))
		})
	})
})
