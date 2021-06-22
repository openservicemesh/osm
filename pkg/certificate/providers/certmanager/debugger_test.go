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
		cert := &Certificate{
			issuingCA:    pem.RootCertificate("zz"),
			privateKey:   pem.PrivateKey("yy"),
			certChain:    pem.Certificate("xx"),
			expiration:   time.Now(),
			commonName:   "foo.bar.co.uk",
			serialNumber: "-certificate-serial-number-",
		}
		cache := map[certificate.CommonName]certificate.Certificater{
			"foo": cert,
		}
		cm := CertManager{
			cache: cache,
		}
		It("lists all issued certificates", func() {
			actual := cm.ListIssuedCertificates()
			expected := []certificate.Certificater{cert}
			Expect(actual).To(Equal(expected))
		})
	})
})
