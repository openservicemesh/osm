package catalog

import (
	"time"

	testclient "k8s.io/client-go/kubernetes/fake"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/service"
)

var _ = Describe("Test certificate tooling", func() {
	namespacedService := service.MeshService{
		Namespace: "namespace-here",
		Name:      "service-name-here",
	}
	mc := NewFakeMeshCatalog(testclient.NewSimpleClientset())

	Context("Testing DecodePEMCertificate along with GetCommonName and IssueCertificate", func() {
		It("issues a PEM certificate with the correct CN", func() {
			cert, err := mc.GetCertificateForService(namespacedService)
			Expect(err).ToNot(HaveOccurred())

			actual := cert.GetCertificateChain()
			expected := "-----BEGIN CERTIFICATE-----\nMII"
			Expect(string(actual[:len(expected)])).To(Equal(expected))

			x509Cert, err := certificate.DecodePEMCertificate(cert.GetCertificateChain())
			Expect(err).ToNot(HaveOccurred())

			expectedCN := "service-name-here.namespace-here.svc.cluster.local"
			Expect(x509Cert.Subject.CommonName).To(Equal(expectedCN))

			Expect(x509Cert.NotAfter.After(time.Now())).To(BeTrue())
			Expect(x509Cert.NotAfter.Before(time.Now().Add(24 * time.Hour))).To(BeTrue())
		})
	})

	Context("Testing GetCertificateForService for issuance and retrieval of cached certificates", func() {
		namespacedService := service.MeshService{
			Namespace: "namespace-here",
			Name:      "service-name-here",
		}
		It("issues a PEM certificate with the correct CN", func() {
			cert, err := mc.GetCertificateForService(namespacedService)
			Expect(err).ToNot(HaveOccurred())

			cachedCert, err := mc.GetCertificateForService(namespacedService)
			Expect(err).ToNot(HaveOccurred())

			Expect(cert).To(Equal(cachedCert))
		})
	})
})
