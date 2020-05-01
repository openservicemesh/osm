package vault

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/hashicorp/vault/api"

	"github.com/open-service-mesh/osm/pkg/certificate"
	"github.com/open-service-mesh/osm/pkg/certificate/providers/tresor/pem"
)

var _ = Describe("Test client helpers", func() {
	Context("Test creating a Certificate from Hashi Vault Secret", func() {
		It("creates a Certificate struct from Hashi Vault Secret struct", func() {

			cn := certificate.CommonName("foo.bar.co.uk")

			secret := &api.Secret{
				Data: map[string]interface{}{
					certificateField: "xx",
					privateKeyField:  "yy",
					issuingCAField:   "zz",
				},
			}

			expiration := time.Now().Add(1 * time.Hour)

			actual := newCert(cn, secret, expiration)

			expected := &Certificate{
				issuingCA:  pem.RootCertificate("zz"),
				privateKey: pem.PrivateKey("yy"),
				certChain:  pem.Certificate("xx"),
				expiration: expiration,
				commonName: "foo.bar.co.uk",
			}

			Expect(actual).To(Equal(expected))
		})
	})
})
