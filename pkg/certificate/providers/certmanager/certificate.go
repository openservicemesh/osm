package certmanager

import (
	"time"

	"github.com/openservicemesh/osm/pkg/certificate"
)

// GetCommonName returns the common name of the given certificate.
func (c Certificate) GetCommonName() certificate.CommonName {
	return c.commonName
}

// GetCertificateChain returns the PEM encoded certificate.
func (c Certificate) GetCertificateChain() []byte {
	return c.certChain
}

// GetPrivateKey returns the PEM encoded private key of the given certificate.
func (c Certificate) GetPrivateKey() []byte {
	return c.privateKey
}

// GetIssuingCA returns the root certificate signing the given cert.
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
