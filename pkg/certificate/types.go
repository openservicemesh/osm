// Package certificate implements utility routines to endcode and decode certificates, and provides the
// interface definitions for Certificate and Certificate Manager.
package certificate

import (
	"sync"
	"time"
<<<<<<< HEAD
=======

	"github.com/openservicemesh/osm/pkg/certificate/pem"
	"github.com/openservicemesh/osm/pkg/messaging"
>>>>>>> caaa189c (feat(certificates) begin to abstract the cert manager patterns (#4580))
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

// SerialNumber is the Serial Number of the given certificate.
type SerialNumber string

func (sn SerialNumber) String() string {
	return string(sn)
}

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

	// GetSerialNumber returns the serial number of the given certificate.
	GetSerialNumber() SerialNumber
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

	// ReleaseCertificate informs the underlying certificate issuer that the given cert will no longer be needed.
	// This method could be called when a given payload is terminated. Calling this should remove certs from cache and free memory if possible.
	ReleaseCertificate(CommonName)
}

type client interface {
	// IssueCertificate issues a new certificate.
	IssueCertificate(CommonName, time.Duration) (*Certificate, error)
}

// manager is a struct that is soon to replace the Manager interface.
// TODO(#4533): export this struct and remove the Manager interface
type manager struct {
	client client

	// The Certificate Authority root certificate to be used by this certificate manager
	ca *Certificate

	// Cache for all the certificates issued
	// Types: map[certificate.CommonName]*certificate.Certificate
	cache sync.Map

	serviceCertValidityDuration time.Duration
	msgBroker                   *messaging.Broker
}
