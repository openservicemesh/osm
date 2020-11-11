package tresor

import (
	"crypto/x509"
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
