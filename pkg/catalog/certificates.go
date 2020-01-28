package catalog

import (
	"github.com/golang/glog"

	"github.com/deislabs/smc/pkg/certificate"
	"github.com/deislabs/smc/pkg/endpoint"
)

func (sc *MeshCatalog) GetCertificateForService(service endpoint.ServiceName) (certificate.Certificater, error) {
	cert, exists := sc.certificateCache[service]
	if exists {
		return cert, nil
	}
	newCert, err := sc.certManager.IssueCertificate(certificate.CommonName(service))
	if err != nil {
		glog.Errorf("Failed issuing a new certificate for service %s: %s", service, err)
		return nil, err
	}
	sc.certificateCache[service] = newCert
	return newCert, nil
}
