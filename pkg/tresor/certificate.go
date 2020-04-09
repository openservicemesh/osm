package tresor

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"time"

	"github.com/pkg/errors"

	"github.com/open-service-mesh/osm/pkg/certificate"
	"github.com/open-service-mesh/osm/pkg/tresor/pem"
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

// GetRootCertificate implements certificate.Certificater and returns the root certificate for the given cert.
func (c Certificate) GetRootCertificate() *x509.Certificate {
	return c.ca
}

// NewCertManagerWithCAFromFile creates a new CertManager with the passed files containing the CA and CA Private Key
func NewCertManagerWithCAFromFile(certFilePEM string, keyFilePEM string, org string, validity time.Duration) (*CertManager, error) {
	ca, _, err := certFromFile(certFilePEM)
	if err != nil {
		return nil, err
	}
	rsaKey, _, err := privKeyFromFile(keyFilePEM)
	if err != nil {
		log.Error().Err(err).Msgf("Error loading private key from file %s", keyFilePEM)
		return nil, err
	}
	return NewCertManagerWithCA(ca, rsaKey, org, validity)
}

// NewCertManagerWithCA creates a new CertManager with the passed CA and CA Private Key
func NewCertManagerWithCA(ca *x509.Certificate, caPrivKey *rsa.PrivateKey, org string, validity time.Duration) (*CertManager, error) {
	cm := CertManager{
		ca:            ca,
		caPrivKey:     caPrivKey,
		announcements: make(chan interface{}),
		org:           org,
		validity:      validity,
		cache:         make(map[certificate.CommonName]Certificate),
	}
	return &cm, nil
}

// NewSelfSignedCert creates a new self-signed certificate.
func NewSelfSignedCert(host string, org string, validity time.Duration) (pem.Certificate, pem.PrivateKey, error) {
	log.Info().Msgf("Generating a new certificate for host: %s", host)
	if host == "" {
		return nil, nil, errInvalidHost
	}
	privateKey, err := rsa.GenerateKey(rand.Reader, rsaBits)
	if err != nil {
		return nil, nil, errors.Wrap(err, errGeneratingPrivateKey.Error())
	}
	template, err := makeTemplate(host, org, validity)
	if err != nil {
		return nil, nil, err
	}
	return genCert(template, template, privateKey, privateKey)
}

func genCert(template, parent *x509.Certificate, certPrivKey, caPrivKey *rsa.PrivateKey) (pem.Certificate, pem.PrivateKey, error) {
	derBytes, err := x509.CreateCertificate(rand.Reader, template, parent, &certPrivKey.PublicKey, caPrivKey)
	if err != nil {
		log.Error().Err(err).Msgf("Error issuing x509.CreateCertificate command for CN=%s", template.Subject.CommonName)
		return nil, nil, errors.Wrap(err, errCreateCert.Error())
	}

	certPEM, err := encodeCert(derBytes)
	if err != nil {
		log.Error().Err(err).Msgf("Error encoding certificate with CN=%s", template.Subject.CommonName)
		return nil, nil, err
	}

	privKeyPEM, err := encodeKey(certPrivKey)
	if err != nil {
		log.Error().Err(err).Msgf("Error encoding private key for certificate with CN=%s", template.Subject.CommonName)
		return nil, nil, err
	}

	return certPEM, privKeyPEM, nil
}
