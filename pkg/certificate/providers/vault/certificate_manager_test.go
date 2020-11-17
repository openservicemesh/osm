package vault

import (
	"net/url"
	"time"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/hashicorp/vault/api"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/certificate/pem"
	"github.com/openservicemesh/osm/pkg/configurator"
)

var _ = Describe("Test client helpers", func() {
	var (
		mockCtrl         *gomock.Controller
		mockConfigurator *configurator.MockConfigurator
	)
	mockCtrl = gomock.NewController(GinkgoT())

	issuingCA := pem.RootCertificate("zz")

	expiredCertCN := certificate.CommonName("this.has.expired")
	expiredCert := &Certificate{
		issuingCA:  pem.RootCertificate("zz"),
		privateKey: pem.PrivateKey("yy"),
		certChain:  pem.Certificate("xx"),
		expiration: time.Now(), // This certificate has ALREADY expired
		commonName: expiredCertCN,
	}

	validCertCN := certificate.CommonName("valid.certificate")
	validCert := &Certificate{
		issuingCA:  issuingCA,
		privateKey: pem.PrivateKey("yy"),
		certChain:  pem.Certificate("xx"),
		expiration: time.Now().Add(24 * time.Hour),
		commonName: validCertCN,
	}

	rootCertCN := certificate.CommonName("root.cert")
	rootCert := &Certificate{
		issuingCA:  pem.RootCertificate("zz"),
		privateKey: pem.PrivateKey("yy"),
		certChain:  pem.Certificate("xx"),
		expiration: time.Now().Add(24 * time.Hour),
		commonName: rootCertCN,
	}

	Context("Test NewCertManager()", func() {
		It("creates new certificate manager", func() {
			vaultAddr := "foo://bar/baz"
			vaultToken := "bar"
			validityPeriod := 1 * time.Second
			vaultRole := "baz"
			mockConfigurator = configurator.NewMockConfigurator(mockCtrl)
			mockConfigurator.EXPECT().GetServiceCertValidityPeriod().Return(validityPeriod).AnyTimes()

			_, err := NewCertManager(vaultAddr, vaultToken, vaultRole, mockConfigurator)
			Expect(err).To(HaveOccurred())
			vaultError := err.(*url.Error)
			expected := `unsupported protocol scheme "foo"`
			Expect(vaultError.Err.Error()).To(Equal(expected))
		})
	})

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

	Context("Test Hashi Vault functions", func() {
		cm := CertManager{
			cache: &map[certificate.CommonName]certificate.Certificater{
				expiredCertCN: expiredCert,
				validCertCN:   validCert,
			},
			ca: rootCert,
		}

		It("gets certs from cache", func() {
			// This cert does not exist - returns nil
			Expect(cm.getFromCache("nothing")).To(BeNil())

			// This cert has expired -- returns nil
			Expect(cm.getFromCache(expiredCertCN)).To(BeNil())

			actual := cm.getFromCache(validCertCN)
			Expect(actual).To(Equal(validCert))
		})

		It("creates certificates", func() {
			actual, err := cm.GetCertificate(validCertCN)
			Expect(err).ToNot(HaveOccurred())
			Expect(actual).To(Equal(validCert))
		})

		It("can fetch root cert", func() {
			actual, err := cm.GetRootCertificate()
			Expect(err).ToNot(HaveOccurred())
			Expect(actual).To(Equal(rootCert))
		})

		It("implements Certificater correctly", func() {
			actual, err := cm.GetCertificate(validCertCN)
			Expect(err).ToNot(HaveOccurred())
			Expect(actual.GetCommonName()).To(Equal(validCertCN))
			Expect(actual.GetCertificateChain()).To(Equal([]byte(validCert.certChain)))
			Expect(actual.GetExpiration()).To(Equal(validCert.expiration))
			Expect(actual.GetIssuingCA()).To(Equal([]byte(issuingCA)))
		})
	})
})
