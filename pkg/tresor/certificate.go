package tresor

import (
	"crypto/x509"
	"time"

	"github.com/open-service-mesh/osm/pkg/certificate"
)

// GetName implements certificate.Certificater and returns the CN of the cert.
func (c Certificate) GetName() string {
	return c.name
}

// GetCertificateChain implements certificate.Certificater and returns the certificate chain.
func (c Certificate) GetCertificateChain() []byte {
	return c.certChain
}

// GetPrivateKey implements certificate.Certificater and returns the private key of the cert.
func (c Certificate) GetPrivateKey() []byte {
	return c.privateKey
}

// GetIssuingCA implements certificate.Certificater and returns the root certificate for the given cert.
func (c Certificate) GetIssuingCA() *x509.Certificate {
	if c.ca == nil {
		log.Info().Msgf("No root certificate available for certificate with CN=%s", c.x509Cert.Subject.CommonName)
		return nil
	}
	return c.ca.x509Cert
}

// LoadCA loads the certificate and its key from the supplied PEM files.
func LoadCA(certFilePEM string, keyFilePEM string) (*Certificate, error) {
	x509Cert, pemCert, err := certFromFile(certFilePEM)
	if err != nil {
		log.Error().Err(err).Msgf("Error loading certificate from file %s", certFilePEM)
		return nil, err
	}

	rsaKey, pemKey, err := privKeyFromFile(keyFilePEM)
	if err != nil {
		log.Error().Err(err).Msgf("Error loading private key from file %s", keyFilePEM)
		return nil, err
	}

	rootCertificate := Certificate{
		name:       rootCertificateName,
		certChain:  pemCert,
		privateKey: pemKey,
		x509Cert:   x509Cert,
		rsaKey:     rsaKey,
		ca:         nil, // this is the CA itself
	}
	return &rootCertificate, nil
}

// NewCertManager creates a new CertManager with the passed CA and CA Private Key
func NewCertManager(ca *Certificate, validity time.Duration) (*CertManager, error) {
	cm := CertManager{
		ca:            ca,
		validity:      validity,
		announcements: make(chan interface{}),
		cache:         make(map[certificate.CommonName]Certificate),
	}
	return &cm, nil
}
