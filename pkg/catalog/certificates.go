package catalog

import (
	"github.com/open-service-mesh/osm/pkg/certificate"
	"github.com/open-service-mesh/osm/pkg/endpoint"
)

// GetCertificateForService returns the certificate the given proxy uses for mTLS to the XDS server.
func (mc *MeshCatalog) GetCertificateForService(service endpoint.NamespacedService) (certificate.Certificater, error) {
	cert, exists := mc.certificateCache[service]
	if exists {
		return cert, nil
	}
	newCert, err := mc.certManager.IssueCertificate(service.GetCommonName())
	if err != nil {
		log.Error().Err(err).Msgf("Error issuing a new certificate for service %s", service)
		return nil, err
	}
	mc.certificateCache[service] = newCert
	return newCert, nil
}
