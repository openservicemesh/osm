package tresor

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Test Certificate Manager", func() {
	Context("Test issuing a certificate", func() {
		validity := 3 * time.Second
		rootCertPem := "sample_certificate.pem"
		rootKeyPem := "sample_private_key.pem"
		cert, err := LoadCA(rootCertPem, rootKeyPem)
		if err != nil {
			log.Fatal().Err(err).Msgf("Error loading CA from files %s and %s", rootCertPem, rootKeyPem)
		}
		m, newCertError := NewCertManager(cert, validity)
		It("should issue a certificate", func() {
			Expect(newCertError).ToNot(HaveOccurred())
			cert, issueCerterror := m.IssueCertificate("a.b.c")
			Expect(issueCerterror).ToNot(HaveOccurred())
			Expect(cert.GetName()).To(Equal("a.b.c"))
		})
	})
})
