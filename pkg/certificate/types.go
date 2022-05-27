// Package certificate implements utility routines to endcode and decode certificates, and provides the
// interface definitions for Certificate and Certificate Manager.
package certificate

import (
	"sync"
	"time"

	"github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"
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

// CertificateType is the type of certificate. This is only used by OSM.
type CertificateType string

func (ct CertificateType) String() string {
	return string(ct)
}

const (
	ADS     CertificateType = "ads"
	XDS     CertificateType = "xds"
	Service CertificateType = "service"
	Ingress CertificateType = "ingress"
	Webhook CertificateType = "webhook"
)

// Certificate represents an x509 certificate.
type Certificate struct {
	// CertificateType is the type of OSM certificate
	CertificateType CertificateType

	// The CommonName of the certificate
	CommonName CommonName

	// The serial number of the certificate
	SerialNumber SerialNumber

	// When the cert expires
	// If this is a composite certificate, the expiration time is the earliest of them.
	Expiration time.Time

	// PEM encoded Certificate and Key (byte arrays)
	CertChain  pem.Certificate
	PrivateKey pem.PrivateKey

	// Certificate authority signing this certificate
	IssuingCA pem.RootCertificate

	keyIssuerID string
	pubIssuerID string
}

// Issuer is the interface for a certificate authority that can issue certificates from a given root certificate.
type Issuer interface {
	// IssueCertificate issues a new certificate.
	IssueCertificate(CommonName, time.Duration) (*Certificate, error)
}

type issuer struct {
	Issuer
	ID string
}

// Manager represents all necessary information for the certificate managers.
type Manager struct {
	// Cache for all the certificates issued
	// Types: map[certificate.CommonName]*certificate.Certificate
	cache sync.Map

	serviceCertValidityDuration func() time.Duration
	msgBroker                   *messaging.Broker

	mu        sync.RWMutex // mu syncrhonizes acces to the below resources.
	keyIssuer *issuer
	pubIssuer *issuer // empty if there is no additional public cert issuer.
}

// MRCClient is an interface that can watch for changes to the MRC. It is typically backed by a k8s informer.
type MRCClient interface {
	List() ([]*v1alpha2.MeshRootCertificate, error)

	// GetCertIssuerForMRC returns an Issuer based on the provided MRC.
	GetCertIssuerForMRC(mrc *v1alpha2.MeshRootCertificate) (Issuer, string, error)
}
