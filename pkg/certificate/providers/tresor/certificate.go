package tresor

import (
	"time"

	"github.com/openservicemesh/osm/pkg/announcements"
	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/certificate/rotor"
	"github.com/openservicemesh/osm/pkg/configurator"
)

const (
	checkCertificateExpirationInterval = 5 * time.Second
)

// GetCommonName implements certificate.Certificater and returns the CN of the cert.
func (c Certificate) GetCommonName() certificate.CommonName {
	return c.commonName
}

// GetCertificateChain implements certificate.Certificater and returns the certificate chain.
func (c Certificate) GetCertificateChain() []byte {
	return c.certChain
}

// GetPrivateKey implements certificate.Certificater and returns the private key of the cert.
func (c Certificate) GetPrivateKey() []byte {
	return c.privateKey
}

// GetIssuingCA implements certificate.Certificater and returns the root certificate for the given cert.
func (c Certificate) GetIssuingCA() []byte {
	return c.issuingCA
}

// GetExpiration implements certificate.Certificater and returns the time the given certificate expires.
func (c Certificate) GetExpiration() time.Time {
	return c.expiration
}

// GetSerialNumber returns the serial number of the given certificate.
func (c Certificate) GetSerialNumber() certificate.SerialNumber {
	return c.serialNumber
}

// NewCertManager creates a new CertManager with the passed CA and CA Private Key
func NewCertManager(ca certificate.Certificater, certificatesOrganization string, cfg configurator.Configurator) (*CertManager, error) {
	if ca == nil {
		return nil, errNoIssuingCA
	}

	certManager := CertManager{
		// The root certificate signing all newly issued certificates
		ca: ca,

		// Channel used to inform other components of cert changes (rotation etc.)
		announcements: make(chan announcements.Announcement),

		certificatesOrganization: certificatesOrganization,

		cfg: cfg,
	}

	// Instantiating a new certificate rotation mechanism will start a goroutine for certificate rotation.
	rotor.New(&certManager).Start(checkCertificateExpirationInterval)

	return &certManager, nil
}
