package tresor

import (
	"time"

	"github.com/openservicemesh/osm/pkg/announcements"
	"github.com/openservicemesh/osm/pkg/certificate/pem"
	"github.com/openservicemesh/osm/pkg/configurator"
)

const (
	rootCertOrganization = "Open Service Mesh Tresor"
)

// NewFakeCertManager creates a fake CertManager used for testing.
func NewFakeCertManager(cfg configurator.Configurator) *CertManager {
	rootCertCountry := "US"
	rootCertLocality := "CA"
	ca, err := NewCA("Fake Tresor CN", 1*time.Hour, rootCertCountry, rootCertLocality, rootCertOrganization)
	if err != nil {
		log.Error().Err(err).Msg("Error creating CA for fake cert manager")
	}

	return &CertManager{
		ca:            ca.(*Certificate),
		announcements: make(chan announcements.Announcement),
		cfg:           cfg,
	}
}

// NewFakeCertificate is a helper creating Certificates for unit tests.
func NewFakeCertificate() *Certificate {
	cert := Certificate{
		privateKey:   pem.PrivateKey("yy"),
		certChain:    pem.Certificate("xx"),
		expiration:   time.Now(),
		commonName:   "foo.bar.co.uk",
		serialNumber: "-the-certificate-serial-number-",
	}

	// It is acceptable in the context of a unit test (so far) for
	// the Issuing CA to be the same as the certificate itself.
	cert.issuingCA = pem.RootCertificate(cert.certChain)

	return &cert
}
