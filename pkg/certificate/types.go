package certificate

import "time"

// CommonName is the Subject Common Name from a given SSL certificate.
type CommonName string

func (cn CommonName) String() string {
	return string(cn)
}

// Certificater is the interface declaring methods each Certificate object must have.
type Certificater interface {

	// GetCommonName retrieves the name of the certificate.
	GetCommonName() string

	// GetCertificateChain retrieves the cert chain.
	GetCertificateChain() []byte

	// GetPrivateKey returns the private key.
	GetPrivateKey() []byte

	// GetIssuingCA returns the root certificate for the given cert.
	GetIssuingCA() []byte

	// GetExpiration returns the time the certificate would expire.
	GetExpiration() time.Time
}

// Manager is the interface declaring the methods for the Certificate Manager.
type Manager interface {
	// IssueCertificate issues a new certificate.
	IssueCertificate(CommonName, *time.Duration) (Certificater, error)

	// RotateCertificate rotates an existing certificate.
	RotateCertificate(CommonName) (Certificater, error)

	// GetRootCertificate returns the root certificate in PEM format and its expiration.
	GetRootCertificate() (Certificater, error)

	// GetAnnouncementsChannel returns a channel, which is used to announce when changes have been made to the issued certificates.
	GetAnnouncementsChannel() <-chan interface{}
}
