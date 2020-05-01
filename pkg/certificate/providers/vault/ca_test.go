package vault

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/hashicorp/vault/api"

	"github.com/open-service-mesh/osm/pkg/certificate"
	"github.com/open-service-mesh/osm/pkg/certificate/providers/tresor/pem"
)

var _ = Describe("Test ca helpers", func() {
	Context("Test creating a root certificate from Hashi Vault Secret", func() {
		It("creates a root certificate struct from Hashi Vault Secret struct", func() {

			commonName := "root-cert.foo.bar.co.uk"
			cn := certificate.CommonName(commonName)

			secret := &api.Secret{
				Data: map[string]interface{}{
					issuingCAField: "zz",
				},
			}

			expiration := time.Now().Add(1 * time.Hour)

			actual := newRootCert(cn, secret, expiration)

			expected := &Certificate{
				issuingCA:  pem.RootCertificate("zz"),
				privateKey: nil,
				certChain:  pem.Certificate("zz"),
				expiration: expiration,
				commonName: certificate.CommonName(commonName),
			}

			Expect(actual).To(Equal(expected))
		})
	})
})
