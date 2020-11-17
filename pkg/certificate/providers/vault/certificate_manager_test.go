package vault

import (
	"net/url"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
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
		It("issues a Certificate struct from Hashi Vault Secret struct", func() {

			cn := certificate.CommonName("foo.bar.co.uk")

			certDuration := 1 * time.Hour

			serNum := uuid.New().String()
			chain := uuid.New().String()
			privKey := uuid.New().String()
			ca := uuid.New().String()

			role := vaultRole("-some-role-")
			mockWrite := func(path string, data map[string]interface{}) (*api.Secret, error) {
				return &api.Secret{
					RequestID:     "",
					LeaseID:       "",
					LeaseDuration: 0,
					Renewable:     false,
					Data: map[string]interface{}{
						serialNumberField: serNum,
						certificateField:  chain,
						privateKeyField:   privKey,
						issuingCAField:    ca,
					},
					Warnings: nil,
					Auth:     nil,
					WrapInfo: nil,
				}, nil
			}

			actual, err := issue(mockWrite, role, cn, certDuration)
			Expect(err).ToNot(HaveOccurred())

			Expect(actual.issuingCA).To(Equal(pem.RootCertificate(ca)))
			Expect(actual.privateKey).To(Equal(pem.PrivateKey(privKey)))
			Expect(actual.certChain).To(Equal(pem.Certificate(chain)))

			Expect(time.Until(actual.expiration)).Should(BeNumerically("<", certDuration))
			Expect(time.Until(actual.expiration)).Should(BeNumerically(">", certDuration-10*time.Second))

			Expect(actual.commonName).To(Equal(certificate.CommonName("foo.bar.co.uk")))
		})
	})

	Context("Test Hashi Vault functions", func() {
		cm := CertManager{
			ca: rootCert,
		}
		cm.cache.Store(expiredCertCN, expiredCert)
		cm.cache.Store(validCertCN, validCert)

		It("gets issuing CA public part", func() {
			expectedNumberOfCertsInCache := 2
			{
				actualCache := make(map[certificate.CommonName]*Certificate)
				cm.cache.Range(func(cnI interface{}, certI interface{}) bool {
					actualCache[cnI.(certificate.CommonName)] = certI.(*Certificate)
					return true
				})
				Expect(len(actualCache)).To(Equal(expectedNumberOfCertsInCache))
			}
			certBytes := uuid.New().String()
			issue := func(hashiVaultWrite, vaultRole, certificate.CommonName, time.Duration) (*Certificate, error) {
				cert := &Certificate{
					issuingCA:    pem.RootCertificate(certBytes),
					serialNumber: "abcd",
				}
				return cert, nil
			}
			issuingCA, err := cm.getIssuingCA(issue)
			Expect(err).ToNot(HaveOccurred())

			// Ensure that cache is NOT affected
			Expect(issuingCA).To(Equal([]byte(certBytes)))
			{
				actualCache := make(map[certificate.CommonName]*Certificate)
				cm.cache.Range(func(cnI interface{}, certI interface{}) bool {
					actualCache[cnI.(certificate.CommonName)] = certI.(*Certificate)
					return true
				})
				Expect(len(actualCache)).To(Equal(expectedNumberOfCertsInCache))
			}

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
