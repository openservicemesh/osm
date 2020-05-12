package catalog

import (
	"github.com/open-service-mesh/osm/pkg/certificate"
	"github.com/open-service-mesh/osm/pkg/service"
)

// GetCertificateForService returns the certificate the given proxy uses for mTLS to the XDS server.
func (mc *MeshCatalog) GetCertificateForService(nsService service.NamespacedService) (certificate.Certificater, error) {
	cert, exists := mc.certificateCache[nsService]
	if exists {
		return cert, nil
	}
	newCert, err := mc.certManager.IssueCertificate(nsService.GetCommonName())
	if err != nil {
		log.Error().Err(err).Msgf("Error issuing a new certificate for service %s", nsService)
		return nil, err
	}
	mc.certificateCache[nsService] = newCert
	return newCert, nil
}
