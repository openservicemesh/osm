package main

import (
	"context"
	"time"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	testclient "k8s.io/client-go/kubernetes/fake"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/certificate/providers/tresor"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/constants"
)

var _ = Describe("Test creation of CA bundle k8s secret", func() {
	var (
		mockCtrl         *gomock.Controller
		mockConfigurator *configurator.MockConfigurator
	)
	mockCtrl = gomock.NewController(GinkgoT())

	Context("Testing createCABundleKubernetesSecret", func() {
		mockConfigurator = configurator.NewMockConfigurator(mockCtrl)
		mockConfigurator.EXPECT().GetServiceCertValidityPeriod().Return(1 * time.Hour).AnyTimes()

		It("creates a k8s secret", func() {

			cache := make(map[certificate.CommonName]certificate.Certificater)
			certManager := tresor.NewFakeCertManager(&cache, mockConfigurator)
			secretName := "--secret--name--"
			namespace := "--namespace--"
			k8sClient := testclient.NewSimpleClientset()

			err := createOrUpdateCABundleKubernetesSecret(k8sClient, certManager, namespace, secretName)
			Expect(err).ToNot(HaveOccurred())

			actual, err := k8sClient.CoreV1().Secrets(namespace).Get(context.Background(), secretName, v1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			expected := "-----BEGIN CERTIFICATE-----\nMIID"
			stringPEM := string(actual.Data[constants.KubernetesOpaqueSecretCAKey])[:len(expected)]
			Expect(stringPEM).To(Equal(expected))

			expectedRootCert, err := certManager.GetRootCertificate()
			Expect(err).ToNot(HaveOccurred())
			Expect(actual.Data[constants.KubernetesOpaqueSecretCAKey]).To(Equal(expectedRootCert.GetCertificateChain()))
		})
	})
})

var _ = Describe("Test joining of URL paths", func() {
	It("should correctly join URL paths", func() {
		final := joinURL("http://foo", "/bar")
		Expect(final).To(Equal("http://foo/bar"))
	})

	It("should correctly join URL paths", func() {
		final := joinURL("http://foo/", "/bar")
		Expect(final).To(Equal("http://foo/bar"))
	})

	It("should correctly join URL paths", func() {
		final := joinURL("http://foo/", "bar")
		Expect(final).To(Equal("http://foo/bar"))
	})

	It("should correctly join URL paths", func() {
		final := joinURL("http://foo", "bar")
		Expect(final).To(Equal("http://foo/bar"))
	})
})
