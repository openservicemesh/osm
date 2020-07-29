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

	Context("Testing encodeExpiration", func() {
		It("serialized expiration date", func() {
			expiration, err := time.Parse(constants.TimeDateLayout, "2020-05-07T14:25:18.677Z")
			Expect(err).ToNot(HaveOccurred())

			actual := encodeExpiration(expiration)
			expected := []byte("MjAyMC0wNS0wN1QxNDoyNToxOC42Nzda")
			Expect(actual).To(Equal(expected))
		})
	})

	Context("Testing getCertFromKubernetes", func() {
		It("obtained root cert from k8s", func() {
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
					constants.KubernetesOpaqueSecretCAExpiration:      []byte("MjAyMC0wNS0wN1QxNDoyNToxOC42Nzda"),
					constants.KubernetesOpaqueSecretRootPrivateKeyKey: keyPEM,
				},
			}

			_, err := kubeClient.CoreV1().Secrets(ns).Create(context.Background(), secret, v1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			actual := getCertFromKubernetes(kubeClient, ns, secretName)

			expectedCert := pem.Certificate(certPEM)
			expectedKey := pem.PrivateKey(keyPEM)
			expiration, err := time.Parse(constants.TimeDateLayout, "2020-05-07T14:25:18.677Z")
			Expect(err).ToNot(HaveOccurred())

			expected, err := tresor.NewCertificateFromPEM(expectedCert, expectedKey, expiration)
			Expect(err).ToNot(HaveOccurred())

			Expect(actual).To(Equal(expected))
		})
	})

	Context("Testing saveSecretToKubernetes", func() {
		It("saves root cert to k8s", func() {
			kubeClient := testclient.NewSimpleClientset()

			ns := uuid.New().String()
			secretName := uuid.New().String()

			certPEM := []byte(uuid.New().String())
			keyPEM := []byte(uuid.New().String())

			expected := &corev1.Secret{
				ObjectMeta: v1.ObjectMeta{
					Name:      secretName,
					Namespace: ns,
				},
				Data: map[string][]byte{
					constants.KubernetesOpaqueSecretCAKey:             certPEM,
					constants.KubernetesOpaqueSecretCAExpiration:      []byte("MjAyMC0wNS0wN1QxNDoyNToxOC42Nzda"),
					constants.KubernetesOpaqueSecretRootPrivateKeyKey: keyPEM,
				},
			}

			expectedCert := pem.Certificate(certPEM)
			expectedKey := pem.PrivateKey(keyPEM)
			expiration, err := time.Parse(constants.TimeDateLayout, "2020-05-07T14:25:18.677Z")
			Expect(err).ToNot(HaveOccurred())
			cert, err := tresor.NewCertificateFromPEM(expectedCert, expectedKey, expiration)
			Expect(err).ToNot(HaveOccurred())

			err = saveSecretToKubernetes(kubeClient, cert, ns, secretName, keyPEM)
			Expect(err).ToNot(HaveOccurred())

			actual, err := kubeClient.CoreV1().Secrets(ns).Get(context.Background(), secretName, v1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			Expect(*actual).To(Equal(*expected))
		})
	})
})
