package vault

import (
	"github.com/openservicemesh/osm/pkg/certificate"
)

// ListIssuedCertificates implements CertificateDebugger interface and returns the list of issued certificates.
func (cm *CertManager) ListIssuedCertificates() []*certificate.Certificate {
	var certs []*certificate.Certificate
	cm.cache.Range(func(cnInterface interface{}, certInterface interface{}) bool {
		certs = append(certs, certInterface.(*certificate.Certificate))
		return true // continue the iteration
	})
	return certs
}
