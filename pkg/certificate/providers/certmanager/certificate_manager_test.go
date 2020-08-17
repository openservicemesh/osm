package certmanager

import (
	"crypto/rand"
	"crypto/x509"
	"time"

	cmapi "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1beta1"
	cmmeta "github.com/jetstack/cert-manager/pkg/apis/meta/v1"
	cmfakeclient "github.com/jetstack/cert-manager/pkg/client/clientset/versioned/fake"
	cmfakeapi "github.com/jetstack/cert-manager/pkg/client/clientset/versioned/typed/certmanager/v1beta1/fake"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/testing"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/logger"
)

var _ = Describe("Test cert-manager Certificate Manager", func() {
	Context("Test Getting a certificate from the cache", func() {
		log := logger.New("cert-manager-test")
		validity := 1 * time.Hour
		rootCertFilePEM := "../../sample_certificate.pem"
		rootKeyFilePEM := "../../sample_private_key.pem"
		cn := certificate.CommonName("bookbuyer.azure.mesh")

		rootCertPEM, err := certificate.LoadCertificateFromFile(rootCertFilePEM)
		if err != nil {
			log.Fatal().Err(err).Msgf("Error loading certificate from file %s", rootCertFilePEM)
		}

		rootCert, err := certificate.DecodePEMCertificate(rootCertPEM)
		if err != nil {
			log.Fatal().Err(err).Msgf("Error decoding certificate from file %s", rootCertFilePEM)
		}
		rootCert.NotAfter = time.Now().Add(time.Minute * 30)

		rootKeyPEM, err := certificate.LoadPrivateKeyFromFile(rootKeyFilePEM)
		if err != nil {
			log.Fatal().Err(err).Msgf("Error loading private ket from file %s", rootCertFilePEM)
		}
		rootKey, err := certificate.DecodePEMPrivateKey(rootKeyPEM)
		if err != nil {
			log.Fatal().Err(err).Msgf("Error decoding private key from file %s", rootKeyFilePEM)
		}

		signedCertDER, err := x509.CreateCertificate(rand.Reader, rootCert, rootCert, rootKey.Public(), rootKey)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to self signed certificate")
		}

		signedCertPEM, err := certificate.EncodeCertDERtoPEM(signedCertDER)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed encode signed signed certificate")
		}

		rootCertificator, err := NewRootCertificateFromPEM(rootCertPEM)
		if err != nil {
			log.Fatal().Err(err).Msgf("Error loading ca %s", rootCertPEM)
		}

		crNotReady := &cmapi.CertificateRequest{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "osm-123",
				Namespace: "osm-system",
			},
		}
		crReady := crNotReady.DeepCopy()
		crReady.Status = cmapi.CertificateRequestStatus{
			Certificate: signedCertPEM,
			CA:          signedCertPEM,
			Conditions: []cmapi.CertificateRequestCondition{
				{
					Type:   cmapi.CertificateRequestConditionReady,
					Status: cmmeta.ConditionTrue,
				},
			},
		}

		fakeClient := cmfakeclient.NewSimpleClientset()
		fakeClient.CertmanagerV1beta1().(*cmfakeapi.FakeCertmanagerV1beta1).Fake.PrependReactor("*", "*", func(action testing.Action) (bool, runtime.Object, error) {
			switch action.GetVerb() {
			case "create":
				return true, crNotReady, nil
			case "get":
				return true, crReady, nil
			case "list":
				return true, &cmapi.CertificateRequestList{Items: []cmapi.CertificateRequest{*crReady}}, nil
			case "delete":
				return true, nil, nil
			default:
				return false, nil, nil
			}
		})

		cm, newCertError := NewCertManager(rootCertificator, fakeClient, "osm-system", validity, cmmeta.ObjectReference{Name: "osm-ca"})
		It("should get an issued certificate from the cache", func() {
			Expect(newCertError).ToNot(HaveOccurred())
			cert, issueCertificateError := cm.IssueCertificate(cn, &validity)
			Expect(issueCertificateError).ToNot(HaveOccurred())
			Expect(cert.GetCommonName()).To(Equal(cn))

			cachedCert, getCertificateError := cm.GetCertificate(cn)
			Expect(getCertificateError).ToNot(HaveOccurred())
			Expect(cachedCert).To(Equal(cert))
		})
	})
})
