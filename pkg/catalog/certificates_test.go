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
			actual := cert.GetCertificateChain()
			expected := []byte{45, 45, 45, 45, 45, 66, 69, 71, 73, 78, 32, 67, 69, 82, 84, 73, 70, 73, 67, 65, 84, 69, 45, 45, 45, 45, 45, 10, 77, 73, 73, 70, 49}
			Expect(actual[:33]).To(Equal(expected))
			x509Cert, err := tresor.DecodePEMCertificate(cert.GetCertificateChain())
			Expect(err).ToNot(HaveOccurred())
			expectedCN := "service-name-here.namespace-here.svc.cluster.local"
			Expect(x509Cert.Subject.CommonName).To(Equal(expectedCN))
		})
	})
})
