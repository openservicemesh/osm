package main

import (
	"context"
	"time"

	"github.com/google/uuid"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	testclient "k8s.io/client-go/kubernetes/fake"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/openservicemesh/osm/pkg/certificate/pem"
	"github.com/openservicemesh/osm/pkg/certificate/providers/tresor"
	"github.com/openservicemesh/osm/pkg/constants"
)

var _ = Describe("Test CMD tools", func() {

	keyPEM := []byte(`-----BEGIN PRIVATE KEY-----
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
`)

	// Source: https://golang.org/src/crypto/x509/example_test.go
	certPEM := []byte(`-----BEGIN CERTIFICATE-----
MIIDujCCAqKgAwIBAgIIE31FZVaPXTUwDQYJKoZIhvcNAQEFBQAwSTELMAkGA1UE
BhMCVVMxEzARBgNVBAoTCkdvb2dsZSBJbmMxJTAjBgNVBAMTHEdvb2dsZSBJbnRl
cm5ldCBBdXRob3JpdHkgRzIwHhcNMTQwMTI5MTMyNzQzWhcNMTQwNTI5MDAwMDAw
WjBpMQswCQYDVQQGEwJVUzETMBEGA1UECAwKQ2FsaWZvcm5pYTEWMBQGA1UEBwwN
TW91bnRhaW4gVmlldzETMBEGA1UECgwKR29vZ2xlIEluYzEYMBYGA1UEAwwPbWFp
bC5nb29nbGUuY29tMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEfRrObuSW5T7q
5CnSEqefEmtH4CCv6+5EckuriNr1CjfVvqzwfAhopXkLrq45EQm8vkmf7W96XJhC
7ZM0dYi1/qOCAU8wggFLMB0GA1UdJQQWMBQGCCsGAQUFBwMBBggrBgEFBQcDAjAa
BgNVHREEEzARgg9tYWlsLmdvb2dsZS5jb20wCwYDVR0PBAQDAgeAMGgGCCsGAQUF
BwEBBFwwWjArBggrBgEFBQcwAoYfaHR0cDovL3BraS5nb29nbGUuY29tL0dJQUcy
LmNydDArBggrBgEFBQcwAYYfaHR0cDovL2NsaWVudHMxLmdvb2dsZS5jb20vb2Nz
cDAdBgNVHQ4EFgQUiJxtimAuTfwb+aUtBn5UYKreKvMwDAYDVR0TAQH/BAIwADAf
BgNVHSMEGDAWgBRK3QYWG7z2aLV29YG2u2IaulqBLzAXBgNVHSAEEDAOMAwGCisG
AQQB1nkCBQEwMAYDVR0fBCkwJzAloCOgIYYfaHR0cDovL3BraS5nb29nbGUuY29t
L0dJQUcyLmNybDANBgkqhkiG9w0BAQUFAAOCAQEAH6RYHxHdcGpMpFE3oxDoFnP+
gtuBCHan2yE2GRbJ2Cw8Lw0MmuKqHlf9RSeYfd3BXeKkj1qO6TVKwCh+0HdZk283
TZZyzmEOyclm3UGFYe82P/iDFt+CeQ3NpmBg+GoaVCuWAARJN/KfglbLyyYygcQq
0SgeDh8dRKUiaW3HQSoYvTvdTuqzwK4CXsr3b5/dAOY8uMuG/IAR3FgwTbZ1dtoW
RvOTa8hYiU6A475WuZKyEHcwnGYe57u2I2KbMgcKjPniocj4QzgYsVAVKW3IwaOh
yE+vPxsiUkvQHdO2fojCkY8jg70jxM+gu59tPDNbw3Uh/2Ij310FgTHsnGQMyA==
-----END CERTIFICATE-----`)

	Context("Testing getCertFromKubernetes", func() {
		It("obtained root cert from k8s", func() {
			kubeClient := testclient.NewSimpleClientset()

			ns := uuid.New().String()
			secretName := uuid.New().String()

			secret := &corev1.Secret{
				ObjectMeta: v1.ObjectMeta{
					Name:      secretName,
					Namespace: ns,
				},
				Data: map[string][]byte{
					constants.KubernetesOpaqueSecretCAKey:             certPEM,
					constants.KubernetesOpaqueSecretCAExpiration:      []byte("2020-05-07T14:25:18.677Z"),
					constants.KubernetesOpaqueSecretRootPrivateKeyKey: keyPEM,
				},
			}

			_, err := kubeClient.CoreV1().Secrets(ns).Create(context.Background(), secret, v1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			actual, err := getCertFromKubernetes(kubeClient, ns, secretName)
			Expect(err).ToNot(HaveOccurred())

			expectedCert := pem.Certificate(certPEM)
			expectedKey := pem.PrivateKey(keyPEM)
			expiration, err := time.Parse(constants.TimeDateLayout, "2020-05-07T14:25:18.677Z")
			Expect(err).ToNot(HaveOccurred())

			expected, err := tresor.NewCertificateFromPEM(expectedCert, expectedKey, expiration)
			Expect(err).ToNot(HaveOccurred())

			Expect(actual).To(Equal(expected))
		})

		It("should not error when the root certificate secret is not found", func() {
			kubeClient := testclient.NewSimpleClientset()

			ns := uuid.New().String()
			secretName := uuid.New().String()

			rootCert, err := getCertFromKubernetes(kubeClient, ns, secretName)
			Expect(err).ToNot(HaveOccurred())
			Expect(rootCert).To(BeNil())
		})

		It("should return an error when the root cert CA is missing in the secret", func() {
			kubeClient := testclient.NewSimpleClientset()

			ns := uuid.New().String()
			secretName := uuid.New().String()

			keyPEM := []byte(uuid.New().String())

			secret := &corev1.Secret{
				ObjectMeta: v1.ObjectMeta{
					Name:      secretName,
					Namespace: ns,
				},
				Data: map[string][]byte{
					constants.KubernetesOpaqueSecretCAExpiration:      []byte("2020-05-07T14:25:18.677Z"),
					constants.KubernetesOpaqueSecretRootPrivateKeyKey: keyPEM,
				},
			}

			_, err := kubeClient.CoreV1().Secrets(ns).Create(context.Background(), secret, v1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			rootCert, err := getCertFromKubernetes(kubeClient, ns, secretName)
			Expect(err).To(HaveOccurred())
			Expect(rootCert).To(BeNil())
		})

		It("should return an error when the root cert private key is missing in the secret", func() {
			kubeClient := testclient.NewSimpleClientset()

			ns := uuid.New().String()
			secretName := uuid.New().String()

			secret := &corev1.Secret{
				ObjectMeta: v1.ObjectMeta{
					Name:      secretName,
					Namespace: ns,
				},
				Data: map[string][]byte{
					constants.KubernetesOpaqueSecretCAKey:        certPEM,
					constants.KubernetesOpaqueSecretCAExpiration: []byte("2020-05-07T14:25:18.677Z"),
				},
			}

			_, err := kubeClient.CoreV1().Secrets(ns).Create(context.Background(), secret, v1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			rootCert, err := getCertFromKubernetes(kubeClient, ns, secretName)
			Expect(err).To(HaveOccurred())
			Expect(rootCert).To(BeNil())
		})

		It("should return an error when the root cert expiration time is missing in the secret", func() {
			kubeClient := testclient.NewSimpleClientset()

			ns := uuid.New().String()
			secretName := uuid.New().String()

			certPEM := []byte(uuid.New().String())
			keyPEM := []byte(uuid.New().String())

			secret := &corev1.Secret{
				ObjectMeta: v1.ObjectMeta{
					Name:      secretName,
					Namespace: ns,
				},
				Data: map[string][]byte{
					constants.KubernetesOpaqueSecretCAKey:             certPEM,
					constants.KubernetesOpaqueSecretRootPrivateKeyKey: keyPEM,
				},
			}

			_, err := kubeClient.CoreV1().Secrets(ns).Create(context.Background(), secret, v1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			rootCert, err := getCertFromKubernetes(kubeClient, ns, secretName)
			Expect(err).To(HaveOccurred())
			Expect(rootCert).To(BeNil())
		})
	})

	Context("Testing saveOrUpdateSecretToKubernetes", func() {
		It("saves root cert to k8s if Secret doesn't exist", func() {
			kubeClient := testclient.NewSimpleClientset()

			ns := uuid.New().String()
			secretName := uuid.New().String()

			expected := &corev1.Secret{
				ObjectMeta: v1.ObjectMeta{
					Name:      secretName,
					Namespace: ns,
				},
				Data: map[string][]byte{
					constants.KubernetesOpaqueSecretCAKey:             certPEM,
					constants.KubernetesOpaqueSecretCAExpiration:      []byte("2020-05-07T14:25:18.677Z"),
					constants.KubernetesOpaqueSecretRootPrivateKeyKey: keyPEM,
				},
			}

			expectedCert := pem.Certificate(certPEM)
			expectedKey := pem.PrivateKey(keyPEM)
			expiration, err := time.Parse(constants.TimeDateLayout, "2020-05-07T14:25:18.677Z")
			Expect(err).ToNot(HaveOccurred())
			cert, err := tresor.NewCertificateFromPEM(expectedCert, expectedKey, expiration)
			Expect(err).ToNot(HaveOccurred())

			err = saveOrUpdateSecretToKubernetes(kubeClient, cert, ns, secretName)
			Expect(err).ToNot(HaveOccurred())

			actual, err := kubeClient.CoreV1().Secrets(ns).Get(context.Background(), secretName, v1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			Expect(*actual).To(Equal(*expected))
		})

		It("updates root cert to k8s if Secret exists", func() {
			ns := uuid.New().String()
			secretName := uuid.New().String()

			kubeClient := testclient.NewSimpleClientset(&corev1.Secret{
				ObjectMeta: v1.ObjectMeta{
					Name:      secretName,
					Namespace: ns,
				},
				Data: map[string][]byte{
					constants.KubernetesOpaqueSecretCAKey:             []byte("abc"),
					constants.KubernetesOpaqueSecretCAExpiration:      []byte("def"),
					constants.KubernetesOpaqueSecretRootPrivateKeyKey: []byte("ghi"),
					"foo": []byte("bar"),
					"123": []byte("456"),
				},
			})

			expected := &corev1.Secret{
				ObjectMeta: v1.ObjectMeta{
					Name:      secretName,
					Namespace: ns,
				},
				Data: map[string][]byte{
					constants.KubernetesOpaqueSecretCAKey:             certPEM,
					constants.KubernetesOpaqueSecretCAExpiration:      []byte("2020-05-07T14:25:18.677Z"),
					constants.KubernetesOpaqueSecretRootPrivateKeyKey: keyPEM,
					"foo": []byte("bar"),
					"123": []byte("456"),
				},
			}

			expectedCert := pem.Certificate(certPEM)
			expectedKey := pem.PrivateKey(keyPEM)
			expiration, err := time.Parse(constants.TimeDateLayout, "2020-05-07T14:25:18.677Z")
			Expect(err).ToNot(HaveOccurred())
			cert, err := tresor.NewCertificateFromPEM(expectedCert, expectedKey, expiration)
			Expect(err).ToNot(HaveOccurred())

			err = saveOrUpdateSecretToKubernetes(kubeClient, cert, ns, secretName)
			Expect(err).ToNot(HaveOccurred())

			actual, err := kubeClient.CoreV1().Secrets(ns).Get(context.Background(), secretName, v1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			Expect(*actual).To(Equal(*expected))
		})
	})
})
