package tresor

import (
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

// TODO(draychev) -- all of these need error output
// GetIssuingCA implements certificate.Certificater and returns the root certificate for the given cert.
func (c Certificate) GetIssuingCA() []byte {
	if c.issuingCA == nil {
		log.Fatal().Msgf("No issuing CA available for cert %s", c.name) // TODO!!!!!!!!!!!!!!!!!
		return nil
	}

	return c.issuingCA.GetCertificateChain()
}

// LoadCA loads the certificate and its key from the supplied PEM files.
func LoadCA(certFilePEM string, keyFilePEM string) (*Certificate, error) {
	pemCert, err := LoadCertificateFromFile(certFilePEM)
	if err != nil {
		log.Error().Err(err).Msgf("Error loading certificate from file %s", certFilePEM)
		return nil, err
	}

	pemKey, err := LoadPrivateKeyFromFile(keyFilePEM)
	if err != nil {
		log.Error().Err(err).Msgf("Error loading private key from file %s", keyFilePEM)
		return nil, err
	}

	rootCertificate := Certificate{
		name:       rootCertificateName,
		certChain:  pemCert,
		privateKey: pemKey,
	}
	return &rootCertificate, nil
}

// NewCertManager creates a new CertManager with the passed CA and CA Private Key
func NewCertManager(ca *Certificate, validity time.Duration) (*CertManager, error) {
	if ca == nil {
		return nil, errNoIssuingCA
	}

	return &CertManager{
		ca:             ca,
		validityPeriod: validity,
		announcements:  make(chan interface{}),
		cache:          make(map[certificate.CommonName]Certificate),
	}, nil
}
