package certificate

import (
	"fmt"
	time "time"

	"github.com/openservicemesh/osm/pkg/certificate/pem"
	"github.com/openservicemesh/osm/pkg/errcode"
)

const (
	// MinRotateBeforeExpireMinutes specifies the minimum number of minutes of how much earlier we can do a certificate renewal.
	// This prevents us from rotating too frequently.
	MinRotateBeforeExpireMinutes = 5

	// Specifies what fraction of validity duration we want to renew before the certificate expires.
	fractionValidityDuration = 3

	// So that we do not renew all certs at the same time - add noise.
	// These define the min and max of the seconds of noise to be added
	// to the early certificate renewal.
	noiseSeconds = 5
)

// mergeRoot will merge in the provided root CA for future calls to GetTrustedCAs. It guarantees to not mutate
// the underlying IssuingCA or trustedCAs fields. By doing so, we ensure that we don't need locks.
// NOTE: this does not return a full copy, mutations to the other byte slices could cause data races.
func (c *Certificate) newMergedWithRoot(root pem.RootCertificate) *Certificate {
	cert := *c

	buf := make([]byte, 0, len(root)+len(c.IssuingCA))
	buf = append(buf, c.IssuingCA...)
	buf = append(buf, root...)
	cert.TrustedCAs = buf
	return &cert
}

func (c *Certificate) String() string {
	return fmt.Sprintf("cert: CommonName: %s, SerialNumber: %s, Expiration: %s", c.CommonName, c.SerialNumber, c.Expiration)
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
	return c.Expiration.Round(0)
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

// GetTrustedCAs returns the PEM-encoded trust context
// for this certificates holder
func (c *Certificate) GetTrustedCAs() pem.RootCertificate {
	return c.TrustedCAs
}

// GetSigningIssuerID returns the signing Issuer ID
// for this certificates holder
func (c *Certificate) GetSigningIssuerID() string {
	return c.signingIssuerID
}

// GetValidatingIssuerID returns the validating Issuer ID
// for this certificates holder
func (c *Certificate) GetValidatingIssuerID() string {
	return c.validatingIssuerID
}

// NewCertificateFromPEM is a helper returning a *certificate.Certificate from the PEM components, signingIssuerID, and validatingIssuerID given
func NewCertificateFromPEM(pemCert, pemKey, caCert []byte,
	signingIssuerID, validatingIssuerID string) (*Certificate, error) {
	x509Cert, err := DecodePEMCertificate(pemCert)
	if err != nil {
		// TODO(#3962): metric might not be scraped before process restart resulting from this error
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrDecodingPEMCert)).
			Msg("Error converting PEM cert to x509 to obtain serial number")
		return nil, err
	}

	return &Certificate{
		CommonName:         CommonName(x509Cert.Subject.CommonName),
		SerialNumber:       SerialNumber(x509Cert.SerialNumber.String()),
		CertChain:          pemCert,
		TrustedCAs:         caCert,
		PrivateKey:         pemKey,
		Expiration:         x509Cert.NotAfter,
		validatingIssuerID: validatingIssuerID,
		signingIssuerID:    signingIssuerID,
		certType:           internal,
	}, nil
}
