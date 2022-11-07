// Package certificate implements utility routines to endcode and decode certificates, and provides the
// interface definitions for Certificate and Certificate Manager.
package certificate

import (
	"context"
	"sync"
	"time"

	"github.com/cskr/pubsub"
	"golang.org/x/sync/singleflight"

	"github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"
	"github.com/openservicemesh/osm/pkg/certificate/pem"
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

// certTypeInternal is the type of certificate. This is only used by OSM.
type certType string

const (
	// internal is the CertType representing all certs issued for use by the OSM
	// control plane.
	internal certType = "internal"

	// ingressGateway is the CertType for certs issued for use by ingress gateways.
	ingressGateway certType = "ingressGateway"

	// service is the CertType for certs issued for use by the data plane.
	service certType = "service"
)

// Certificate represents an x509 certificate.
type Certificate struct {
	// The CommonName of the certificate
	CommonName CommonName

	cacheKey string

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

	certType certType
}

// Issuer is the interface for a certificate authority that can issue certificates from a given root certificate.
type Issuer interface {
	// IssueCertificate issues a new certificate.
	IssueCertificate(IssueOptions) (*Certificate, error)
}

type issuer struct {
	Issuer
	ID            string
	TrustDomain   string
	SpiffeEnabled bool
	// memoized once the first certificate is issued
	CertificateAuthority pem.RootCertificate
}

// Manager represents all necessary information for the certificate managers.
type Manager struct {
	// cache for all the certificates issued
	// Types: map[certificate.CommonName]*certificate.Certificate
	cache sync.Map

	mrcClient MRCClient

	ingressCertValidityDuration func() time.Duration
	// TODO(#4711): define serviceCertValidityDuration in the MRC
	serviceCertValidityDuration func() time.Duration

	mu            sync.Mutex // mu syncrhonizes access to the below resources.
	signingIssuer *issuer
	// equal to signingIssuer if there is no additional public cert issuer.
	validatingIssuer *issuer

	group singleflight.Group

	pubsub *pubsub.PubSub
}

// MRCClient is an interface that can watch for changes to the MRC. It is typically backed by a k8s informer.
type MRCClient interface {
	UpdateMeshRootCertificate(mrc *v1alpha2.MeshRootCertificate) error
	ListMeshRootCertificates() ([]*v1alpha2.MeshRootCertificate, error)
	MRCEventBroker

	// GetCertIssuerForMRC returns an Issuer based on the provided MRC.
	GetCertIssuerForMRC(mrc *v1alpha2.MeshRootCertificate) (Issuer, pem.RootCertificate, error)
}

// MRCEventType is a type alias for a string describing the type of MRC event
type MRCEventType string

// MRCEvent describes a change event on a given MRC
type MRCEvent struct {
	// The name of the MRC generating the event
	MRCName string
}

// MRCEventBroker describes any type that allows the caller to Watch() MRCEvents
type MRCEventBroker interface {
	// Watch allows the caller to subscribe to events surrounding
	// MRCs. Watch returns a channel that emits events, and
	// an error if the subscription goes awry.
	Watch(context.Context) (<-chan MRCEvent, error)
}
