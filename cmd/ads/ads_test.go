package main

import (
	"context"

	"github.com/open-service-mesh/osm/pkg/constants"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-service-mesh/osm/pkg/tresor"
	testclient "k8s.io/client-go/kubernetes/fake"
)

var _ = Describe("Test creation of CA bundle k8s secret", func() {
	Context("Testing createCABundleKubernetesSecret", func() {
		It("creates a k8s secret", func() {

			certManager := tresor.NewFakeCertManager()
			secretName := "--secret--name--"
			namespace := "--namespace--"
			k8sClient := testclient.NewSimpleClientset()

			err := createCABundleKubernetesSecret(k8sClient, certManager, namespace, secretName)
			Expect(err).ToNot(HaveOccurred())

			actual, err := k8sClient.CoreV1().Secrets(namespace).Get(context.Background(), secretName, v1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			expected := "-----BEGIN CERTIFICATE-----\nMIIF"
			stringPEM := string(actual.Data[constants.KubernetesOpaqueSecretCAKey])[:len(expected)]
			Expect(stringPEM).To(Equal(expected))
			Expect(len(actual.Data[constants.KubernetesOpaqueSecretCAKey])).To(Equal(1915))
		})
	})
})
