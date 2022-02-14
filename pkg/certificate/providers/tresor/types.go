// Package tresor implements the certificate.Manager interface for Tresor, a custom certificate provider in OSM.
package tresor

import (
	"math/big"
	"sync"
	"time"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/logger"
	"github.com/openservicemesh/osm/pkg/messaging"
)

const (
	// How many bits to use for the RSA key
	rsaBits = 2048

	// How many bits in the certificate serial number
	certSerialNumberBits = 128
)

var (
	log               = logger.New("tresor")
	serialNumberLimit = new(big.Int).Lsh(big.NewInt(1), certSerialNumberBits)
)

// CertManager implements certificate.Manager
type CertManager struct {
	// The Certificate Authority root certificate to be used by this certificate manager
	ca *certificate.Certificate

	// Cache for all the certificates issued
	// Types: map[certificate.CommonName]*certificate.Certificate
	cache sync.Map

	certificatesOrganization string

	cfg configurator.Configurator

	serviceCertValidityDuration time.Duration
	keySize                     int

	msgBroker *messaging.Broker
}
