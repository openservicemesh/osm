// Package certmanager implements the certificate.Manager interface for cert-manager.io as the certificate provider.
package certmanager

import (
	cmmeta "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	cmclient "github.com/cert-manager/cert-manager/pkg/client/clientset/versioned/typed/certmanager/v1"
	cmlisters "github.com/cert-manager/cert-manager/pkg/client/listers/certmanager/v1"

	"github.com/openservicemesh/osm/pkg/logger"
)

var (
	log = logger.New("cert-manager")
)

// CertManager implements certificate.Manager
type CertManager struct {
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
