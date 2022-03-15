// Package certmanager implements the certificate.Manager interface for cert-manager.io as the certificate provider.
package certmanager

import (
	cmmeta "github.com/jetstack/cert-manager/pkg/apis/meta/v1"
	cmclient "github.com/jetstack/cert-manager/pkg/client/clientset/versioned/typed/certmanager/v1"
	cmlisters "github.com/jetstack/cert-manager/pkg/client/listers/certmanager/v1"

	"github.com/openservicemesh/osm/pkg/certificate"
<<<<<<< HEAD
	"github.com/openservicemesh/osm/pkg/certificate/pem"
	"github.com/openservicemesh/osm/pkg/configurator"
=======
>>>>>>> caaa189c (feat(certificates) begin to abstract the cert manager patterns (#4580))
	"github.com/openservicemesh/osm/pkg/logger"
)

var (
	log = logger.New("cert-manager")
)

// CertManager implements certificate.Manager
type CertManager struct {
	// The Certificate Authority root certificate to be used by this certificate
	// manager.
	ca certificate.Certificater

<<<<<<< HEAD
	// cache holds a local cache of issued certificates as
	// certificate.Certificaters
	cache     map[certificate.CommonName]certificate.Certificater
	cacheLock sync.RWMutex

=======
>>>>>>> caaa189c (feat(certificates) begin to abstract the cert manager patterns (#4580))
	// Control plane namespace where CertificateRequests are created.
	namespace string

	// cert-manager CertificateRequest client set.
	client cmclient.CertificateRequestInterface

	// Reference to the Issuer to sign certificates.
	issuerRef cmmeta.ObjectReference

	// crLister is used to list CertificateRequests in the given namespace.
	crLister cmlisters.CertificateRequestNamespaceLister

	// Issuing certificate properties.
	keySize int
}

// Certificate implements certificate.Certificater
type Certificate struct {
	// The commonName of the certificate
	commonName certificate.CommonName

	// The serial number of the certificate
	serialNumber certificate.SerialNumber

	// When the cert expires
	expiration time.Time

	// PEM encoded Certificate and Key (byte arrays)
	certChain  pem.Certificate
	privateKey pem.PrivateKey

	// Certificate authority signing this certificate.
	issuingCA pem.RootCertificate
}
