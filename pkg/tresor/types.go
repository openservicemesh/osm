package tresor

import (
	"math/big"
	"time"

	"github.com/open-service-mesh/osm/pkg/certificate"
	"github.com/open-service-mesh/osm/pkg/logger"
	"github.com/open-service-mesh/osm/pkg/tresor/pem"
)

const (
	// TypeCertificate is a string constant to be used in the generation of a certificate.
	TypeCertificate = "CERTIFICATE"

	// TypePrivateKey is a string constant to be used in the generation of a private key for a certificate.
	TypePrivateKey = "PRIVATE KEY"

	// String constant used for the commonName of the root certificate
	rootCertificateName = "root-certificate"

	// How many bits to use for the RSA key
	rsaBits = 4096

	// How many bits in the certificate serial number
	certSerialNumberBits = 128

	// Organization field of certificates issued by Tresor
	org = "Open Service Mesh Tresor"
)

var (
	log               = logger.New("tresor")
	serialNumberLimit = new(big.Int).Lsh(big.NewInt(1), certSerialNumberBits)
)

// CertManager implements certificate.Manager
type CertManager struct {
	// How long will newly issued certificates be valid for
	validityPeriod time.Duration

	// The Certificate Authority root certificate to be used by this certificate manager
	ca *Certificate

	// The channel announcing to the rest of the system when a certificate has changed
	announcements <-chan interface{}

	// Cache for all the certificates issued
	cache map[certificate.CommonName]Certificate
}

// Certificate implements certificate.Certificater
type Certificate struct {
	// The commonName of the certificate
	commonName certificate.CommonName

	// TODO(draychev): implement!
	// When the cert expires
	expiration time.Time

	// PEM encoded Certificate and Key (byte arrays)
	certChain  pem.Certificate
	privateKey pem.PrivateKey

	// The CA issuing this certificate.
	// If the certificate itself is a root certificate this would be nil.
	issuingCA *Certificate
}
