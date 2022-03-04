package tresor

import (
	"time"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/certificate/pem"
)

const (
	rootCertOrganization = "Open Service Mesh"
)

// NewFakeProvider creates a fake CertManager used for testing.
func NewFakeProvider() *Provider {
	rootCertCountry := "US"
	rootCertLocality := "CA"
	ca, err := NewCA("Fake Provider CN", 1*time.Hour, rootCertCountry, rootCertLocality, rootCertOrganization)
	if err != nil {
		log.Error().Err(err).Msg("Error creating CA for fake cert manager")
	}
	return &Provider{
		ca:      ca,
		keySize: 2048, // hardcoding this to remove depdendency on configurator mock
	}
}

// NewFakeCertificate is a helper creating Certificates for unit tests.
func NewFakeCertificate() *certificate.Certificate {
	return &certificate.Certificate{
		PrivateKey:   pem.PrivateKey("yy"),
		CertChain:    pem.Certificate("xx"),
		IssuingCA:    pem.RootCertificate("xx"),
		Expiration:   time.Now(),
		CommonName:   "foo.bar.co.uk",
		SerialNumber: "-the-certificate-serial-number-",
	}
}
