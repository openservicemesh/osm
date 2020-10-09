package tresor

import (
	"math/big"
	"sync"
	"time"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/certificate/pem"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/logger"
)

const (
	// String constant used for the commonName of the root certificate
	rootCertificateName = "root-certificate"

	// How many bits to use for the RSA key
	rsaBits = 4096

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
	ca certificate.Certificater

	// The channel announcing to the rest of the system when a certificate has changed
	announcements chan interface{}

	// Cache for all the certificates issued
	cache     *map[certificate.CommonName]certificate.Certificater
	cacheLock sync.Mutex

	certificatesOrganization string

	cfg configurator.Configurator
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

	// Certificate authority signing this certificate
	issuingCA pem.RootCertificate
}
