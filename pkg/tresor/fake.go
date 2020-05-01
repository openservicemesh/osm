package tresor

import (
	"time"

	"github.com/open-service-mesh/osm/pkg/certificate"
)

// NewFakeCertManager creates a fake CertManager used for testing.
func NewFakeCertManager() *CertManager {
	ca, err := NewCA("Fake Tresor CN", 1*time.Hour)
	if err != nil {
		log.Error().Err(err).Msg("Error creating CA for fake cert manager")
	}
	return &CertManager{
		ca:             ca.(*Certificate),
		validityPeriod: 1 * time.Hour,
		announcements:  make(chan interface{}),
		cache:          make(map[certificate.CommonName]Certificate),
	}
}
