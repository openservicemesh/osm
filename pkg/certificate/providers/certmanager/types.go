// Package certmanager implements the certificate.Manager interface for cert-manager.io as the certificate provider.
package certmanager

import (
	"time"

	cmmeta "github.com/jetstack/cert-manager/pkg/apis/meta/v1"
	cmclient "github.com/jetstack/cert-manager/pkg/client/clientset/versioned/typed/certmanager/v1"
	cmlisters "github.com/jetstack/cert-manager/pkg/client/listers/certmanager/v1"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/logger"
)

const (
	// checkCertificateExpirationInterval is the interval to check whether a
	// certificate is close to expiration and needs renewal.
	checkCertificateExpirationInterval = 5 * time.Second
)

var (
	log = logger.New("cert-manager")
)

// CertManager implements certificate.Manager
type Provider struct {
	// The Certificate Authority root certificate to be used by this certificate
	// manager.
	ca *certificate.Certificate

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

// CertManagerOptions is a type that specifies 'cert-manager.io' certificate provider options
type Options struct {
	IssuerName  string
	IssuerKind  string
	IssuerGroup string
}
