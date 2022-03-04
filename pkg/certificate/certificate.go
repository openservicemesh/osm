package certificate

import (
	time "time"

	"github.com/openservicemesh/osm/pkg/certificate/pem"
	"github.com/openservicemesh/osm/pkg/errcode"
	"github.com/rs/zerolog/log"
)

// NewFromPEM is a helper returning a *certificate.Certificate from the PEM components given.
func NewFromPEM(pemCert pem.Certificate, pemKey pem.PrivateKey) (*Certificate, error) {
	x509Cert, err := DecodePEMCertificate(pemCert)
	if err != nil {
		// TODO(#3962): metric might not be scraped before process restart resulting from this error
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrDecodingPEMCert)).
			Msg("Error converting PEM cert to x509 to obtain serial number")
		return nil, err
	}

	return &Certificate{
		CommonName:   CommonName(x509Cert.Subject.CommonName),
		SerialNumber: SerialNumber(x509Cert.SerialNumber.String()),
		CertChain:    pemCert,
		IssuingCA:    pem.RootCertificate(pemCert),
		PrivateKey:   pemKey,
		Expiration:   x509Cert.NotAfter,
	}, nil
}

// GetCommonName returns the Common Name of the certificate
func (c *Certificate) GetCommonName() CommonName {
	return c.CommonName
}

// GetSerialNumber returns the serial number of the certificate
func (c *Certificate) GetSerialNumber() SerialNumber {
	return c.SerialNumber
}

// GetExpiration returns the expiration time of the certificate
func (c *Certificate) GetExpiration() time.Time {
	return c.Expiration
}

// GetCertificateChain returns the certificate chain of the certificate
func (c *Certificate) GetCertificateChain() pem.Certificate {
	return c.CertChain
}

// GetPrivateKey returns the private key of the certificate
func (c *Certificate) GetPrivateKey() pem.PrivateKey {
	return c.PrivateKey
}

// GetIssuingCA returns the issuing CA of the certificate
func (c *Certificate) GetIssuingCA() pem.RootCertificate {
	return c.IssuingCA
}
