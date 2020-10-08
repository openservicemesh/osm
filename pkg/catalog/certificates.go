package catalog

import (
	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/service"
)

// GetCertificateForService returns the certificate the given proxy uses for mTLS to the XDS server.
func (mc *MeshCatalog) GetCertificateForService(meshService service.MeshService) (certificate.Certificater, error) {
	cn := meshService.GetCommonName()

	cert, err := mc.certManager.GetCertificate(cn)
	if err != nil {
		// Certificate was not found in CertManager's cache, issue one
		newCert, err := mc.certManager.IssueCertificate(cn, mc.configurator.GetServiceCertValidityPeriod())
		if err != nil {
			log.Error().Err(err).Msgf("Error issuing a new certificate for service:%s, CN: %s", meshService, cn)
			return nil, err
		}
		return newCert, nil
	}
	return cert, nil
}
