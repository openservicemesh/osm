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
	"github.com/openservicemesh/osm/pkg/certificate/providers"
	"github.com/openservicemesh/osm/pkg/certificate/providers/tresor"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/tests"
)

var _ = Describe("Test CMD tools", func() {
	certPEM, err := tests.GetPEMCert()
	It("should have resulted in no errors", func() {
		Expect(err).ToNot(HaveOccurred())
	})

	keyPEM, err := tests.GetPEMPrivateKey()
	It("should have resulted in no errors", func() {
		Expect(err).ToNot(HaveOccurred())
	})

	Context("Testing getCertFromKubernetes", func() {
		It("obtained root cert from k8s", func() {
			kubeClient := testclient.NewSimpleClientset()

			ns := uuid.New().String()
			secretName := uuid.New().String()

			certProviderConfig := providers.NewCertificateProviderConfig(kubeClient, nil, nil, providers.Kind(certProviderKind), ns,
				secretName, tresorOptions, vaultOptions, certManagerOptions)
			Expect(err).ToNot(HaveOccurred())

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

			actual, err := certProviderConfig.GetCertFromKubernetes()
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

			certProviderConfig := providers.NewCertificateProviderConfig(kubeClient, nil, nil, providers.Kind(certProviderKind), ns,
				secretName, tresorOptions, vaultOptions, certManagerOptions)
			Expect(err).ToNot(HaveOccurred())

			rootCert, err := certProviderConfig.GetCertFromKubernetes()
			Expect(err).ToNot(HaveOccurred())
			Expect(rootCert).To(BeNil())
		})

		It("should return an error when the root cert CA is missing in the secret", func() {
			kubeClient := testclient.NewSimpleClientset()

			ns := uuid.New().String()
			secretName := uuid.New().String()

			certProviderConfig := providers.NewCertificateProviderConfig(kubeClient, nil, nil, providers.Kind(certProviderKind), ns,
				secretName, tresorOptions, vaultOptions, certManagerOptions)
			Expect(err).ToNot(HaveOccurred())

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

			rootCert, err := certProviderConfig.GetCertFromKubernetes()
			Expect(err).To(HaveOccurred())
			Expect(rootCert).To(BeNil())
		})

		It("should return an error when the root cert private key is missing in the secret", func() {
			kubeClient := testclient.NewSimpleClientset()

			ns := uuid.New().String()
			secretName := uuid.New().String()

			certProviderConfig := providers.NewCertificateProviderConfig(kubeClient, nil, nil, providers.Kind(certProviderKind), ns,
				secretName, tresorOptions, vaultOptions, certManagerOptions)
			Expect(err).ToNot(HaveOccurred())

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

			rootCert, err := certProviderConfig.GetCertFromKubernetes()
			Expect(err).To(HaveOccurred())
			Expect(rootCert).To(BeNil())
		})

		It("should return an error when the root cert expiration time is missing in the secret", func() {
			kubeClient := testclient.NewSimpleClientset()

			ns := uuid.New().String()
			secretName := uuid.New().String()

			certProviderConfig := providers.NewCertificateProviderConfig(kubeClient, nil, nil, providers.Kind(certProviderKind), ns,
				secretName, tresorOptions, vaultOptions, certManagerOptions)
			Expect(err).ToNot(HaveOccurred())

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

			rootCert, err := certProviderConfig.GetCertFromKubernetes()
			Expect(err).To(HaveOccurred())
			Expect(rootCert).To(BeNil())
		})
	})
})
