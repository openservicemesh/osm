// Package certificate implements utility routines to endcode and decode certificates, and provides the
// interface definitions for Certificate and Certificate Manager.
package certificate

import (
	"sync"
	"time"

	"github.com/openservicemesh/osm/pkg/certificate/pem"
	"github.com/openservicemesh/osm/pkg/messaging"
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

// Certificate represents an x509 certificate.
type Certificate struct {
	// The CommonName of the certificate
	CommonName CommonName

	// The serial number of the certificate
	SerialNumber SerialNumber

	// When the cert expires
	Expiration time.Time

	// PEM encoded Certificate and Key (byte arrays)
	CertChain  pem.Certificate
	PrivateKey pem.PrivateKey

	// Certificate authority signing this certificate
	IssuingCA pem.RootCertificate
}

type client interface {
	// IssueCertificate issues a new certificate.
	IssueCertificate(CommonName, time.Duration) (*Certificate, error)

	// GetRootCertificate returns the root certificate.
	GetRootCertificate() *Certificate
}

// Manager represents all necessary information for the certificate managers.
type Manager struct {
	clients []client

	// Cache for all the certificates issued
	// Types: map[certificate.CommonName]*certificate.Certificate
	cache sync.Map

	serviceCertValidityDuration time.Duration
	msgBroker                   *messaging.Broker
}
