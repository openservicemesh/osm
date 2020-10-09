package certmanager

import (
	"crypto/rand"
	"crypto/x509"
	"time"

	"github.com/golang/mock/gomock"
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
	"github.com/openservicemesh/osm/pkg/configurator"
)

var _ = Describe("Test cert-manager Certificate Manager", func() {
	defer GinkgoRecover()

	var (
		mockCtrl         *gomock.Controller
		mockConfigurator *configurator.MockConfigurator
	)

	mockCtrl = gomock.NewController(GinkgoT())
	mockConfigurator = configurator.NewMockConfigurator(mockCtrl)

	Context("Test Getting a certificate from the cache", func() {
		validity := 1 * time.Hour
		rootCertFilePEM := "../../sample_certificate.pem"
		rootKeyFilePEM := "../../sample_private_key.pem"
		cn := certificate.CommonName("bookbuyer.azure.mesh")

		rootCertPEM, err := certificate.LoadCertificateFromFile(rootCertFilePEM)
		if err != nil {
			GinkgoT().Fatalf("Error loading certificate from file %s: %s", rootCertFilePEM, err.Error())
		}

		rootCert, err := certificate.DecodePEMCertificate(rootCertPEM)
		if err != nil {
			GinkgoT().Fatalf("Error decoding certificate from file %s: %s", rootCertFilePEM, err.Error())
		}
		rootCert.NotAfter = time.Now().Add(time.Minute * 30)

		rootKeyPEM, err := certificate.LoadPrivateKeyFromFile(rootKeyFilePEM)
		if err != nil {
			GinkgoT().Fatalf("Error loading private ket from file %s: %s", rootCertFilePEM, err.Error())
		}
		rootKey, err := certificate.DecodePEMPrivateKey(rootKeyPEM)
		if err != nil {
			GinkgoT().Fatalf("Error decoding private key from file %s: %s", rootKeyFilePEM, err.Error())
		}

		signedCertDER, err := x509.CreateCertificate(rand.Reader, rootCert, rootCert, rootKey.Public(), rootKey)
		if err != nil {
			GinkgoT().Fatalf("Failed to self signed certificate: %s", err.Error())
		}

		signedCertPEM, err := certificate.EncodeCertDERtoPEM(signedCertDER)
		if err != nil {
			GinkgoT().Fatalf("Failed encode signed signed certificate: %s", err.Error())
		}

		rootCertificator, err := NewRootCertificateFromPEM(rootCertPEM)
		if err != nil {
			GinkgoT().Fatalf("Error loading ca %s: %s", rootCertPEM, err.Error())
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

		cm, newCertError := NewCertManager(rootCertificator, fakeClient, "osm-system", cmmeta.ObjectReference{Name: "osm-ca"}, mockConfigurator)
		It("should get an issued certificate from the cache", func() {
			Expect(newCertError).ToNot(HaveOccurred())
			cert, issueCertificateError := cm.IssueCertificate(cn, validity)
			Expect(issueCertificateError).ToNot(HaveOccurred())
			Expect(cert.GetCommonName()).To(Equal(cn))

			cachedCert, getCertificateError := cm.GetCertificate(cn)
			Expect(getCertificateError).ToNot(HaveOccurred())
			Expect(cachedCert).To(Equal(cert))
		})
	})
})
