package vault

import (
	"net/url"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/hashicorp/vault/api"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/certificate/pem"
	"github.com/openservicemesh/osm/pkg/configurator"

	tassert "github.com/stretchr/testify/assert"
)

var (
	issuingCA = pem.RootCertificate("zz")

	expiredCertCN = certificate.CommonName("this.has.expired")
	expiredCert   = &certificate.Certificate{
		IssuingCA:    pem.RootCertificate("zz"),
		PrivateKey:   pem.PrivateKey("yy"),
		CertChain:    pem.Certificate("xx"),
		Expiration:   time.Now(), // This certificate has ALREADY expired
		CommonName:   expiredCertCN,
		SerialNumber: "-serial-number-",
	}

	validCertCN = certificate.CommonName("valid.certificate")
	validCert   = &certificate.Certificate{
		IssuingCA:    issuingCA,
		PrivateKey:   pem.PrivateKey("yy"),
		CertChain:    pem.Certificate("xx"),
		Expiration:   time.Now().Add(24 * time.Hour),
		CommonName:   validCertCN,
		SerialNumber: "-serial-number-",
	}

	rootCertCN = certificate.CommonName("root.cert")
	rootCert   = &certificate.Certificate{
		IssuingCA:    pem.RootCertificate("zz"),
		PrivateKey:   pem.PrivateKey("yy"),
		CertChain:    pem.Certificate("xx"),
		Expiration:   time.Now().Add(24 * time.Hour),
		CommonName:   rootCertCN,
		SerialNumber: "-serial-number-",
	}
)

var _ = Describe("Test client helpers", func() {
	var (
		mockCtrl         *gomock.Controller
		mockConfigurator *configurator.MockConfigurator
	)
	mockCtrl = gomock.NewController(GinkgoT())

	Context("Test NewCertManager()", func() {
		It("creates new certificate manager", func() {
			vaultAddr := "foo://bar/baz"
			vaultToken := "bar"
			validityPeriod := 1 * time.Second
			vaultRole := "baz"
			mockConfigurator = configurator.NewMockConfigurator(mockCtrl)
			mockConfigurator.EXPECT().GetServiceCertValidityPeriod().Return(validityPeriod).AnyTimes()

			_, err := NewCertManager(
				vaultAddr,
				vaultToken,
				vaultRole,
				mockConfigurator,
				mockConfigurator.GetServiceCertValidityPeriod(),
				nil,
			)
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
					certificateField:  "xx",
					privateKeyField:   "yy",
					issuingCAField:    "zz",
					serialNumberField: "123",
				},
			}

			expiration := time.Now().Add(1 * time.Hour)

			actual := newCert(cn, secret, expiration)

			expected := &certificate.Certificate{
				IssuingCA:    pem.RootCertificate("zz"),
				PrivateKey:   pem.PrivateKey("yy"),
				CertChain:    pem.Certificate("xx"),
				Expiration:   expiration,
				CommonName:   "foo.bar.co.uk",
				SerialNumber: "123",
			}

			Expect(actual).To(Equal(expected))
		})
	})

	Context("Test Hashi Vault functions", func() {
		cm := CertManager{
			ca: rootCert,
		}
		cm.cache.Store(expiredCertCN, expiredCert)
		cm.cache.Store(validCertCN, validCert)

		getCachedCertificateCNs := func() []certificate.CommonName {
			var commonNames []certificate.CommonName
			cm.cache.Range(func(cnInterface interface{}, certInterface interface{}) bool {
				cert := certInterface.(*certificate.Certificate)
				commonNames = append(commonNames, cert.GetCommonName())
				return true // continue the iteration
			})
			return commonNames
		}

		It("gets issuing CA public part", func() {
			certSerialNum := certificate.SerialNumber(uuid.New().String())
			expectedNumberOfCertsInCache := 2
			Expect(len(getCachedCertificateCNs())).To(Equal(expectedNumberOfCertsInCache))
			Expect(getCachedCertificateCNs()).To(ContainElement(certificate.CommonName("this.has.expired")))
			Expect(getCachedCertificateCNs()).To(ContainElement(certificate.CommonName("valid.certificate")))
			certBytes := uuid.New().String()
			issue := func(certificate.CommonName, time.Duration) (*certificate.Certificate, error) {
				cert := &certificate.Certificate{
					IssuingCA:    pem.RootCertificate(certBytes),
					SerialNumber: certSerialNum,
				}
				return cert, nil
			}
			issuingCA, serialNumber, err := cm.getIssuingCA(issue)
			Expect(err).ToNot(HaveOccurred())

			// Ensure that cache is NOT affected
			Expect(issuingCA).To(Equal([]byte(certBytes)))
			Expect(len(getCachedCertificateCNs())).To(Equal(expectedNumberOfCertsInCache))
			Expect(getCachedCertificateCNs()).To(ContainElement(certificate.CommonName("this.has.expired")))
			Expect(getCachedCertificateCNs()).To(ContainElement(certificate.CommonName("valid.certificate")))
			Expect(serialNumber).To(Equal(certSerialNum))
		})

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

		It("implements Certificate correctly", func() {
			actual, err := cm.GetCertificate(validCertCN)
			Expect(err).ToNot(HaveOccurred())
			Expect(actual.CommonName).To(Equal(validCertCN))
			Expect([]byte(actual.GetCertificateChain())).To(Equal([]byte(validCert.GetCertificateChain())))
			Expect(actual.GetExpiration()).To(Equal(validCert.GetExpiration()))
			Expect([]byte(actual.IssuingCA)).To(Equal([]byte(issuingCA)))
		})
	})
})

func TestListCertificiates(t *testing.T) {
	assert := tassert.New(t)

	cm := CertManager{
		ca: rootCert,
	}

	emptyCertList, err := cm.ListCertificates()
	assert.Nil(err)
	assert.Equal(0, len(emptyCertList))

	cm.cache.Store(expiredCertCN, expiredCert)
	cm.cache.Store(validCertCN, validCert)
	certList, err := cm.ListCertificates()
	assert.Nil(err)
	assert.Equal(2, len(certList))
	assert.Contains(certList, expiredCert)
	assert.Contains(certList, validCert)
}
