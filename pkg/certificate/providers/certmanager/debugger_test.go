package certmanager

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/certificate/pem"
)

var _ = Describe("Test cert-manager Debugger", func() {
	Context("test ListIssuedCertificates()", func() {
		cert := &certificate.Certificate{
			IssuingCA:    pem.RootCertificate("zz"),
			PrivateKey:   pem.PrivateKey("yy"),
			CertChain:    pem.Certificate("xx"),
			Expiration:   time.Now(),
			CommonName:   "foo.bar.co.uk",
			SerialNumber: "-certificate-serial-number-",
		}
		cache := map[certificate.CommonName]*certificate.Certificate{
			"foo": cert,
		}
		cm := CertManager{
			cache: cache,
		}
		It("lists all issued certificates", func() {
			actual := cm.ListIssuedCertificates()
			expected := []*certificate.Certificate{cert}
			Expect(actual).To(Equal(expected))
		})
	})
})
