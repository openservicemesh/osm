package tresor

import (
	"time"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/certificate/pem"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/messaging"
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
		ca:      ca,
		cfg:     cfg,
		keySize: 2048, // hardcoding this to remove depdendency on configurator mock
	}
}

// NewFakeCertManagerForRotation creates a fake CertManager used for testing certificate rotation
func NewFakeCertManagerForRotation(cfg configurator.Configurator, msgBroker *messaging.Broker) *CertManager {
	cm := NewFakeCertManager(cfg)
	cm.msgBroker = msgBroker
	return cm
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
