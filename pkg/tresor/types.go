package tresor

import (
	"crypto/rsa"
	"crypto/x509"
	"reflect"
	"time"

	zlog "github.com/rs/zerolog/log"

	"github.com/open-service-mesh/osm/pkg/certificate"
	"github.com/open-service-mesh/osm/pkg/utils"
)

const (
	// TypeCertificate is a string constant to be used in the generation of a certificate.
	TypeCertificate = "CERTIFICATE"

	// TypePrivateKey is a string constant to be used in the generation of a private key for a certificate.
	TypePrivateKey = "PRIVATE KEY"
)

var (
	packageName = utils.GetLastChunkOfSlashed(reflect.TypeOf(empty{}).PkgPath())
	log         = zlog.With().Str("comp", packageName).Caller().Logger()
)

// CertManager implements certificate.Manager
type CertManager struct {
	ca            *x509.Certificate
	caPrivKey     *rsa.PrivateKey
	announcements <-chan interface{}
	org           string
	validity      time.Duration
	cache         map[certificate.CommonName]Certificate
}

// Certificate implements certificate.Certificater
type Certificate struct {
	name       string
	certChain  []byte
	privateKey []byte
	ca         *x509.Certificate
}
