package certificate

import (
	"time"

	"github.com/openservicemesh/osm/pkg/logger"
)

const (
	// TypeCertificate is a string constant to be used in the generation of a certificate.
	TypeCertificate = "CERTIFICATE"

	// TypePrivateKey is a string constant to be used in the generation of a private key for a certificate.
	TypePrivateKey = "PRIVATE KEY"

	// TypeCertificateRequest is a string constant to be used in the generation
	// of a certificate requests.
	TypeCertificateRequest = "CERTIFICATE REQUEST"
)

// CommonName is the Subject Common Name from a given SSL certificate.
type CommonName string

func (cn CommonName) String() string {
	return string(cn)
}

// Certificater is the interface declaring methods each Certificate object must have.
type Certificater interface {

	// GetCommonName retrieves the name of the certificate.
	GetCommonName() CommonName

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
	IssueCertificate(CommonName, time.Duration) (Certificater, error)

	// GetCertificate returns a certificate given its Common Name (CN)
	GetCertificate(CommonName) (Certificater, error)

	// RotateCertificate rotates an existing certificate.
	RotateCertificate(CommonName) (Certificater, error)

	// GetRootCertificate returns the root certificate in PEM format and its expiration.
	GetRootCertificate() (Certificater, error)

	// ListCertificates lists all certificates issued
	ListCertificates() ([]Certificater, error)

	// GetAnnouncementsChannel returns a channel, which is used to announce when changes have been made to the issued certificates.
	GetAnnouncementsChannel() <-chan interface{}
}

var (
	log = logger.New("certificate")
)
