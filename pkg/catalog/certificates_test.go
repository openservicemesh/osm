package catalog

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/open-service-mesh/osm/pkg/endpoint"
	"github.com/open-service-mesh/osm/pkg/ingress"
	"github.com/open-service-mesh/osm/pkg/smi"
	"github.com/open-service-mesh/osm/pkg/tresor"
)

func newMeshCatalog() *MeshCatalog {
	meshSpec := smi.NewFakeMeshSpecClient()
	certManager := tresor.NewFakeCertManager()
	ingressMonitor := ingress.NewFakeIngressMonitor()
	stop := make(<-chan struct{})
	var endpointProviders []endpoint.Provider
	return NewMeshCatalog(meshSpec, certManager, ingressMonitor, stop, endpointProviders...)
}

var _ = Describe("Test certificate tooling", func() {
	Context("Testing DecodePEMCertificate along with GetCommonName and IssueCertificate", func() {
		mc := newMeshCatalog()
		It("issues a PEM certificate with the correct CN", func() {
			namespacedService := endpoint.NamespacedService{
				Namespace: "namespace-here",
				Service:   "service-name-here",
			}

			cert, err := mc.certManager.IssueCertificate(namespacedService.GetCommonName())
			Expect(err).ToNot(HaveOccurred())

			actual := cert.GetCertificateChain()
			expected := "-----BEGIN CERTIFICATE-----\nMII"
			Expect(string(actual[:len(expected)])).To(Equal(expected))

			x509Cert, err := tresor.DecodePEMCertificate(cert.GetCertificateChain())
			Expect(err).ToNot(HaveOccurred())

			expectedCN := "service-name-here.namespace-here.svc.cluster.local"
			Expect(x509Cert.Subject.CommonName).To(Equal(expectedCN))

			Expect(x509Cert.NotAfter.After(time.Now())).To(BeTrue())
			Expect(x509Cert.NotAfter.Before(time.Now().Add(24 * time.Hour))).To(BeTrue())
		})
	})
})
