package catalog

import (
	"github.com/golang/glog"

	"github.com/open-service-mesh/osm/pkg/certificate"
	"github.com/open-service-mesh/osm/pkg/endpoint"
)

// GetCertificateForService returns the certificate the given proxy uses for mTLS to the XDS server.
func (sc *MeshCatalog) GetCertificateForService(service endpoint.NamespacedService) (certificate.Certificater, error) {
	cert, exists := sc.certificateCache[service]
	if exists {
		return cert, nil
	}
	newCert, err := sc.certManager.IssueCertificate(certificate.CommonName(service.String()))
	if err != nil {
		glog.Errorf("Failed issuing a new certificate for service %s: %s", service, err)
		return nil, err
	}
	sc.certificateCache[service] = newCert
	return newCert, nil
}
