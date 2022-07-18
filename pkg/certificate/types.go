// Package certificate implements utility routines to endcode and decode certificates, and provides the
// interface definitions for Certificate and Certificate Manager.
package certificate

import (
	"context"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"

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

// CertType is the type of certificate. This is only used by OSM.
type CertType string

const (
	// Internal is the CertType representing all certs issued for use by the OSM
	// control plane.
	Internal CertType = "internal"

	// IngressGateway is the CertType for certs issued for use by ingress gateways.
	IngressGateway CertType = "ingressGateway"

	// Service is the CertType for certs issued for use by the data plane.
	Service CertType = "service"
)

// Certificate represents an x509 certificate.
type Certificate struct {
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

	// Certificate Authority signing this certificate
	IssuingCA pem.RootCertificate

	// The trust context of this certificate's recipient
	// Includes both issuing CA and validating CA (if applicable)
	TrustedCAs pem.RootCertificate

	signingIssuerID    string
	validatingIssuerID string

	certType CertType
}

// Issuer is the interface for a certificate authority that can issue certificates from a given root certificate.
type Issuer interface {
	// IssueCertificate issues a new certificate.
	IssueCertificate(CommonName, time.Duration) (*Certificate, error)
}

type issuer struct {
	Issuer
	ID          string
	TrustDomain string
	// memoized once the first certificate is issued
	CertificateAuthority pem.RootCertificate
}

// Manager represents all necessary information for the certificate managers.
type Manager struct {
	// Cache for all the certificates issued
	// Types: map[certificate.CommonName]*certificate.Certificate
	cache sync.Map

	ingressCertValidityDuration func() time.Duration
	// TODO(#4711): define serviceCertValidityDuration in the MRC
	serviceCertValidityDuration func() time.Duration
	msgBroker                   *messaging.Broker

	mu            sync.Mutex // mu syncrhonizes acces to the below resources.
	signingIssuer *issuer
	// equal to signingIssuer if there is no additional public cert issuer.
	validatingIssuer *issuer

	group singleflight.Group
}

// MRCClient is an interface that can watch for changes to the MRC. It is typically backed by a k8s informer.
type MRCClient interface {
	List() ([]*v1alpha2.MeshRootCertificate, error)
	MRCEventBroker

	// GetCertIssuerForMRC returns an Issuer based on the provided MRC.
	GetCertIssuerForMRC(mrc *v1alpha2.MeshRootCertificate) (Issuer, pem.RootCertificate, error)
}

// MRCEventType is a type alias for a string describing the type of MRC event
type MRCEventType string

// MRCEvent describes a change event on a given MRC
type MRCEvent struct {
	Type MRCEventType
	// The last observed version of the MRC as of the time of this event
	MRC *v1alpha2.MeshRootCertificate
}

var (
	// MRCEventAdded is the type of announcement emitted when we observe an addition of a Kubernetes MeshRootCertificate
	MRCEventAdded MRCEventType = "meshrootcertificate-added"

	// MRCEventUpdated is the type of announcement emitted when we observe an update to a Kubernetes MeshRootCertificate
	MRCEventUpdated MRCEventType = "meshrootcertificate-updated"
)

// MRCEventBroker describes any type that allows the caller to Watch() MRCEvents
type MRCEventBroker interface {
	// Watch allows the caller to subscribe to events surrounding
	// MRCs. Watch returns a channel that emits events, and
	// an error if the subscription goes awry.
	Watch(context.Context) (<-chan MRCEvent, error)
}
