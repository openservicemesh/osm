package certmanager

import (
	"sync"
	"time"

	cmmeta "github.com/jetstack/cert-manager/pkg/apis/meta/v1"
	cmclient "github.com/jetstack/cert-manager/pkg/client/clientset/versioned/typed/certmanager/v1beta1"
	cmlisters "github.com/jetstack/cert-manager/pkg/client/listers/certmanager/v1beta1"
	"github.com/rs/zerolog"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/certificate/pem"
)

const (
	// How many bits to use for the RSA key
	rsaBits = 4096

	// checkCertificateExpirationInterval is the interval to check wether a
	// certificate is close to expiration and needs renewal.
	checkCertificateExpirationInterval = 5 * time.Second
)

// CertManager implements certificate.Manager
type CertManager struct {
	log zerolog.Logger

	// How long will newly issued certificates be valid for.
	validityPeriod time.Duration

	// The Certificate Authority root certificate to be used by this certificate
	// manager.
	ca certificate.Certificater

	// cache holds a local cache of issued certificates as
	// certificate.Certificaters
	cache     map[certificate.CommonName]certificate.Certificater
	cacheLock sync.RWMutex

	// The channel announcing to the rest of the system when a certificate has
	// changed.
	announcements chan interface{}

	certificatesOrganization string

	// Control plane namespace where CertificateRequests are created.
	namespace string

	// cert-manager CertificateRequest client set.
	client cmclient.CertificateRequestInterface

	// Reference to the Issuer to sign certificates.
	issuerRef cmmeta.ObjectReference

	// crLister is used to list CertificateRequests in the given namespace.
	crLister cmlisters.CertificateRequestNamespaceLister
}

// Certificate implements certificate.Certificater
type Certificate struct {
	// The commonName of the certificate
	commonName certificate.CommonName

	// When the cert expires
	expiration time.Time

	// PEM encoded Certificate and Key (byte arrays)
	certChain  pem.Certificate
	privateKey pem.PrivateKey

	// Certificate authority signing this certificate.
	issuingCA pem.RootCertificate
}
