package vault

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/certificate/pem"
)

var _ = Describe("Test Vault Debugger", func() {
	Context("test ListIssuedCertificates()", func() {
		cert := &certificate.Certificate{
			IssuingCA:    pem.RootCertificate("zz"),
			PrivateKey:   pem.PrivateKey("yy"),
			CertChain:    pem.Certificate("xx"),
			Expiration:   time.Now(),
			CommonName:   "foo.bar.co.uk",
			SerialNumber: "-cert-serial-number-",
		}
		cm := CertManager{}
		cm.cache.Store("foo", cert)
		It("lists all issued certificates", func() {
			actual := cm.ListIssuedCertificates()
			expected := []*certificate.Certificate{cert}
			Expect(actual).To(Equal(expected))
		})
	})
})
