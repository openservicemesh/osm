package tresor

import (
	"crypto/rsa"
	"crypto/x509"
	"time"

	"github.com/deislabs/smc/pkg/certificate"
)

type CA []byte
type CAPrivateKey []byte
type CertPEM []byte
type CertPrivKeyPEM []byte

const (
	TypeCertificate = "CERTIFICATE"
	TypePrivateKey  = "PRIVATE KEY"
)

// Implements certificate.Manager
type CertManager struct {
	ca            *x509.Certificate
	caPrivKey     *rsa.PrivateKey
	announcements <-chan interface{}
	org           string
	validity      time.Duration
	cache         map[certificate.CommonName]Certificate
}

// Implements certificate.Certificater
type Certificate struct {
	name       string
	certChain  []byte
	privateKey []byte
}
