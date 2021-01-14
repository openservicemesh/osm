package certmanager

import (
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/golang/mock/gomock"
	cmapi "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1beta1"
	cmmeta "github.com/jetstack/cert-manager/pkg/apis/meta/v1"
	cmfakeclient "github.com/jetstack/cert-manager/pkg/client/clientset/versioned/fake"
	cmfakeapi "github.com/jetstack/cert-manager/pkg/client/clientset/versioned/typed/certmanager/v1beta1/fake"
	tresorPem "github.com/openservicemesh/osm/pkg/certificate/pem"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/testing"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/configurator"
)

var errDecodingPEMBlock = errors.New("failed to decode PEM block containing certificate")

func getPEMCert() (tresorPem.Certificate, error) {
	// Source: https://golang.org/src/crypto/x509/example_test.go
	const sampleCertificatePEM = `-----BEGIN CERTIFICATE-----
MIIElzCCA3+gAwIBAgIRAOsakgIV4yyGqyI9QlXD6l4wDQYJKoZIhvcNAQELBQAw
WTE4MDYGA1UEAwwvNjNkMDQ0YzktNzdjNy00MmFlLWFmZGMtNjM2YTFiNmFiNGUy
LmF6dXJlLm1lc2gxEDAOBgNVBAoMB0F6IE1lc2gxCzAJBgNVBAYTAlVTMB4XDTIw
MDIxMjAyMjc0OVoXDTIwMDIxMjAyMjg0OVowMTEQMA4GA1UEChMHQUNNRSBDbzEd
MBsGA1UEAxMUYm9va2J1eWVyLmF6dXJlLm1lc2gwggIiMA0GCSqGSIb3DQEBAQUA
A4ICDwAwggIKAoICAQDEvG2HsGCMPXp2ZIkETP5XXuTpmaxBfVBoq3i60Dlun01i
BE8mc3YkUZyJrxqYNCmqc9lqiDboNOs79lRXF47V9pk9l1vgKSnWWrB+AYKuWR2M
g2ZT8QkhYc3lxqpIaL9RqmhmXISID1B2SsaJ6BBEZxXD1EaH3EZza7B5MdzU7CJc
S7ZMV9mlECCv3Ea0+/dWjsLVw5kk75VN8UcSEX10q5weDKnktSoNfM/FJG4rrBTE
CEkdZpiI9rkFl3UvfJ7YaxhDmudyvwxsWWITOFM7uMZkBv9DyIQq1yRh4qI++/K/
aNZhZyYwrbAu0x8xbtkJIleA/gwKDdvBJusu2Qr+xqHghhspXpkH4KE0Tv8xDe0u
6RJkl7eIaUe4cO0q+WTZR10CjvirEp/PFBEUiXFrms+aw6aSRUVFyX7fTHhvc9Pc
jg+5Q72qaRIyyJ65BKO4ZRsTAzqTBlHMyr6HCIKD/Ms0OJ5PPuyKLd6uSG1gGk1e
+LbRpZNpNtNn/EV7x3cMr8zNHA4cdsdWdYkn8zm0pnSK1r+GmmmDP6L+QfvKwxoT
kr0rsvzAecsFfu5E+b+qNqcVoRxxazUR2td8pwovMtuw/H9BreDEbXsQ9xTAig1v
f1+Br2yRZyHn1Zs16Brb9t4QXQDe8ybV8T/Okeh8EHSzRxOkS+LU0XqF0QXJawID
AQABo4GBMH8wDgYDVR0PAQH/BAQDAgWgMB0GA1UdJQQWMBQGCCsGAQUFBwMCBggr
BgEFBQcDATAMBgNVHRMBAf8EAjAAMB8GA1UdIwQYMBaAFGUkH8DwtMhY+Ik26Hwp
utG4F2tgMB8GA1UdEQQYMBaCFGJvb2tidXllci5henVyZS5tZXNoMA0GCSqGSIb3
DQEBCwUAA4IBAQDDUNyGRMNQz6mUlE0K+hKtv3apsW1QL885259xBP41zL3luqMa
kFvKhaNvBDmnPUBmG14bcT/eJE9/WaQc2NgE68WB3Uup4gDxIonmKih9BD4+kjw2
c/jJmWL4dTt3NA3kU/JoTSy+EPMnOnFk88djYmBF1u50QH1PS47r7FWDkhYw7qAn
CJCGDhwLD968m0RWYmeRNMw4d8pFXw8mLl2EbN00k5y8dEnX2P0ciS7LgY/EmIHH
KmPaHP5NIaW0a5riM1iXw9AzBEIfZ8hPJF+qt05v8aRBixD+ufVd/7cgm686EoeO
tvggl4Pyw9tGpQrvZ3tL2HanuRyHiTwXZVzx
-----END CERTIFICATE-----`

	caBlock, _ := pem.Decode([]byte(sampleCertificatePEM))
	if caBlock == nil || caBlock.Type != certificate.TypeCertificate {
		return nil, errDecodingPEMBlock
	}

	return pem.EncodeToMemory(caBlock), nil
}

// getPEMPrivateKey loads a private key from a PEM file.
func getPEMPrivateKey() (tresorPem.PrivateKey, error) {
	const samplePrivateKeyPEM = `-----BEGIN PRIVATE KEY-----
MIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQDD3+gqR5tLq3w2
KZOVCJRaQ2+0bdDmqvWf4YZjsYlIWUMSxQNhX9fm6u/X/fUbwVMpDP3t2A7ArgJP
iakti8676Ws7utVbYi2PvjLfcVtsM0UBtAqXfHN2Rg+Ne7B9AanepUeJIfzs+/jr
6MAhuhTZA/RquhLbRGJKrmHsgnGuAyGn581TXiL52HUvbJ89BbexpcQtUnqFUj8J
hnHWKTuoNPcLlDMRL5fRX08Zyhzxiyg66ALoZduHNu6HV/Z0YXHlxePKZCIRrbx5
8a74q6zYBTWdWqkKhKF1wFYWBwi2ppIPW2U47TOV0IsnWs9o7DsWkFMpf97SpE7v
SyxpPefNAgMBAAECggEATsKJp/aDCzo5B85P+W0pueHD2NkPVrEHcvJMB2oruVur
DLELWuwe9EsjhcYn+LETrz36HNjzlaZiZ3kC/b1ps0V4SNwnTkd76oCgFBiQmkFD
ThwG5kK0aqphNpK1tI4mr8/lo8521RO8U5+TIfygxWJBtWh8jI5Ct6TG20LYUw9a
QMhgmEFVXaBRyoIhccuWahJHSwZzlxlmLTj06Gf+Uv9Snhwy7LJe81i9CNWVn8E0
zW+77vUWQ1/AXIyh0fLmQhisHs6d/wbVr9E8GBAyyzN21uzoXNSyWxnwlGk/K1IQ
76KrRVw7zIQ7iqrEsycMtY8uoW8CkRHZOYvtAS5OQQKBgQD4IllwZRbiWFaRXN04
bUgiFjBQjkCMKyPk1b9MryaG4kIgxN9YQRiwwFWueaW4p+HyujT8pAl4xo5RbH37
xKPqgPCQ1XzH9mPo7Mx0OCyv9GaAXlq4FqiJU5T5xF6SoWSgJTKgVPfNtGLAzWaX
l/BRY+19ATAL1kSRXKq7cHpJjwKBgQDKFXZpq5QPXk37CE1hpN6cs8cKkvfU4oaq
V4lC+4TlAah8JjtzXNyAbKtGdV9Q9kgsgDBeaTBY4MZrtnhh6JVY3twGaRBq6pcv
0IleaVVhp7eOwMA4W5AYSnZ6LahFY0YFyzFeEgyzqwbQlFX+A9ovXX+DJlBoM6pn
gcowfqNy4wKBgAVs8tmzTCnM1q+9ARVPxmkAZTQNuDmYY+OIDPPHTKdcYSfIRj3u
xnRu8DCtdkMwYI9nJOt1RsO+S7RaE/MiXJcvFJOGJ4FT0OFx9BKCe++o/2jFJ2Sp
EixWiIZhldPM9Z9O0OmSkgyMajBfDWQ5LUcKUVIPaZaIq90l0pHgprvfAoGBALBc
eMIR3p5m8/FQNpAv3aOuddfxmV5t74675GvTrBBcGRl4GEw+z6U4sWVFS9ERjr1f
hlbuwCXgzOn2DiuMWsJ7hFQH3y8f2p/9A9WkYcJfJ5/q8hZ9Ok0otys7q24bDGJE
CaqKYBFxAfqIal/MJt9NXtorVuMJq/63U6hs7OJ3AoGAAz5s2BEJQ4V5eD3U2ybn
pxtNBGA9nxmM8LZlg80XdhBfrWp44rCPOWsZEUlI800gy3qerF1bZywpWkDydJrX
TDO2ZGgoxQvaQfdAhjYKeD+7/Y9M/AacQSDaYOeXAdR9f6hJrf+1SHAGjqbaUXuR
sIpZJboKv7uhHDhGJsdP/8Y=
-----END PRIVATE KEY-----
`

	caKeyBlock, _ := pem.Decode([]byte(samplePrivateKeyPEM))
	if caKeyBlock == nil || caKeyBlock.Type != certificate.TypePrivateKey {
		return nil, errDecodingPEMBlock
	}

	return pem.EncodeToMemory(caKeyBlock), nil
}

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

		cn := certificate.CommonName("bookbuyer.azure.mesh")

		rootCertPEM, err := getPEMCert()
		if err != nil {
			GinkgoT().Fatalf("Error loading sample test certificate: %s", err.Error())
		}

		rootCert, err := certificate.DecodePEMCertificate(rootCertPEM)
		if err != nil {
			GinkgoT().Fatalf("Error decoding certificate from file: %s", err.Error())
		}
		Expect(rootCert).ToNot(BeNil())
		rootCert.NotAfter = time.Now().Add(time.Minute * 30)

		rootKeyPEM, err := getPEMPrivateKey()
		if err != nil {
			GinkgoT().Fatalf("Error loading private key: %s", err.Error())
		}
		rootKey, err := certificate.DecodePEMPrivateKey(rootKeyPEM)
		if err != nil {
			GinkgoT().Fatalf("Error decoding private key: %s", err.Error())
		}
		Expect(rootKey).ToNot(BeNil())

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
