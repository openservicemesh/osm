// Package certmanager implements the certificate.Manager interface for cert-manager.io as the certificate provider.
package certmanager

import (
	"sync"
	"time"

	cmmeta "github.com/jetstack/cert-manager/pkg/apis/meta/v1"
	cmclient "github.com/jetstack/cert-manager/pkg/client/clientset/versioned/typed/certmanager/v1"
	cmlisters "github.com/jetstack/cert-manager/pkg/client/listers/certmanager/v1"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/logger"
	"github.com/openservicemesh/osm/pkg/messaging"
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
type CertManager struct {
	// The Certificate Authority root certificate to be used by this certificate
	// manager.
	ca *certificate.Certificate

	// cache holds a local cache of issued certificates as
	// *certificate.Certificates
	cache     map[certificate.CommonName]*certificate.Certificate
	cacheLock sync.RWMutex

	// Control plane namespace where CertificateRequests are created.
	namespace string

	// cert-manager CertificateRequest client set.
	client cmclient.CertificateRequestInterface

	// Reference to the Issuer to sign certificates.
	issuerRef cmmeta.ObjectReference

	// crLister is used to list CertificateRequests in the given namespace.
	crLister cmlisters.CertificateRequestNamespaceLister

	cfg configurator.Configurator

	// Issuing certificate properties.
	serviceCertValidityDuration time.Duration
	keySize                     int

	msgBroker *messaging.Broker
}
