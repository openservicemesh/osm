package tresor

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Test Certificate Manager", func() {
	Context("Test issuing a certificate", func() {
		validity := 3 * time.Second
		org := "contoso"
		m, err1 := NewCertManagerWithCAFromFile("sample_certificate.pem", "sample_private_key.pem", org, validity)
		It("should issue a certificate", func() {
			Expect(err1).ToNot(HaveOccurred())
			cert, err2 := m.IssueCertificate("a.b.c")
			Expect(err2).ToNot(HaveOccurred())
			Expect(cert.GetName()).To(Equal("a.b.c"))
		})
	})
})
