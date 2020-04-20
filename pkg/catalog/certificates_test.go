package catalog

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/open-service-mesh/osm/pkg/endpoint"
	"github.com/open-service-mesh/osm/pkg/smi"
	"github.com/open-service-mesh/osm/pkg/tresor"
)

func newMeshCatalog() *MeshCatalog {
	meshSpec := smi.NewFakeMeshSpecClient()
	certManager := tresor.NewFakeCertManager()
	stop := make(<-chan struct{})
	var endpointProviders []endpoint.Provider
	return NewMeshCatalog(meshSpec, certManager, stop, endpointProviders...)
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

			expected := "-----BEGIN CERTIFICATE-----\nMIIF2jCCA8KgAw"
			Expect(string(actual[:len(expected)])).To(Equal(expected))

			x509Cert, err := tresor.DecodePEMCertificate(cert.GetCertificateChain())
			Expect(err).ToNot(HaveOccurred())

			expectedCN := "service-name-here.namespace-here.svc.cluster.local"
			Expect(x509Cert.Subject.CommonName).To(Equal(expectedCN))
		})
	})
})
