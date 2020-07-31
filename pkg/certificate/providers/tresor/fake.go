package tresor

import (
	"time"

	"github.com/openservicemesh/osm/pkg/certificate"
)

// NewFakeCertManager creates a fake CertManager used for testing.
func NewFakeCertManager(cache *map[certificate.CommonName]certificate.Certificater, validityPeriod time.Duration) *CertManager {
	rootCertCountry := "US"
	rootCertLocality := "CA"
	rootCertOrganization := "Open Service Mesh Tresor"
	ca, err := NewCA("Fake Tresor CN", 1*time.Hour, rootCertCountry, rootCertLocality, rootCertOrganization)
	if err != nil {
		log.Error().Err(err).Msg("Error creating CA for fake cert manager")
	}

	return &CertManager{
		ca:             ca.(*Certificate),
		validityPeriod: validityPeriod,
		announcements:  make(chan interface{}),
		cache:          cache,
	}
}
