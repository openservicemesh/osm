package certmanager

import (
	"github.com/openservicemesh/osm/pkg/certificate"
)

// ListIssuedCertificates implements CertificateDebugger interface and returns the list of issued certificates.
func (cm *CertManager) ListIssuedCertificates() []certificate.Certificater {
	cm.cacheLock.RLock()
	defer cm.cacheLock.RUnlock()

	var certs []certificate.Certificater
	for _, cert := range cm.cache {
		certs = append(certs, cert)
	}

	return certs
}
