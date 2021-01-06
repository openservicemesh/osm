package tresor

import "github.com/openservicemesh/osm/pkg/certificate"

// ListIssuedCertificates implements CertificateDebugger interface and returns the list of issued certificates.
func (cm *CertManager) ListIssuedCertificates() []certificate.Certificater {
	var certs []certificate.Certificater
	cm.cache.Range(func(cnInterface interface{}, certInterface interface{}) bool {
		certs = append(certs, certInterface.(certificate.Certificater))
		return true // continue the iteration
	})
	return certs
}
