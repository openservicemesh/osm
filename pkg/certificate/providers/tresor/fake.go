package tresor

import (
	"time"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/certificate/pem"
	"github.com/openservicemesh/osm/pkg/messaging"
)

const (
	rootCertOrganization = "Open Service Mesh Tresor"
)

// NewFake constructs a fake certificate client using a certificate
func NewFake(msgBroker *messaging.Broker) certificate.Manager {
	rootCertCountry := "US"
	rootCertLocality := "CA"
	ca, err := NewCA("Fake Tresor CN", 1*time.Hour, rootCertCountry, rootCertLocality, rootCertOrganization)
	if err != nil {
		log.Error().Err(err).Msg("Error creating CA for fake cert manager")
	}
	tresorClient, err := New(ca, rootCertOrganization, 2048)
	if err != nil {
		log.Error().Err(err).Msg("Error creating CA for fake cert manager")
	}
	tresorCertManager, err := certificate.NewManager(ca, tresorClient, 1*time.Hour, msgBroker)
	if err != nil {
		log.Error().Err(err).Msg("error instantiating osm certificate.Manager for Tresor cert-manager")
	}
	return tresorCertManager
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
