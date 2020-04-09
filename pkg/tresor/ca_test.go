package tresor

import (
	"crypto/x509"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Test creation of a new CA", func() {
	Context("Create a new CA", func() {
		cert, err := NewCA(2 * time.Second)
		It("should create a new CA", func() {
			Expect(err).ToNot(HaveOccurred())
			Expect(cert.x509Cert.NotAfter.Sub(cert.x509Cert.NotBefore)).To(Equal(2 * time.Second))
			Expect(cert.x509Cert.KeyUsage).To(Equal(x509.KeyUsageCertSign | x509.KeyUsageCRLSign))
			Expect(cert.x509Cert.IsCA).To(BeTrue())
		})
	})
})
