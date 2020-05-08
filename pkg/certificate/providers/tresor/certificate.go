package tresor

import (
	"time"

	"github.com/open-service-mesh/osm/pkg/certificate"
	"github.com/open-service-mesh/osm/pkg/certificate/rotor"
)

const (
	checkCertificateExpirationInterval = 5 * time.Second
)

// GetName implements certificate.Certificater and returns the CN of the cert.
func (c Certificate) GetName() string {
	return c.commonName.String()
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
func (c Certificate) GetIssuingCA() []byte {
	if c.issuingCA == nil {
		log.Fatal().Msgf("No issuing CA available for cert %s", c.commonName)
		return nil
	}

	return c.issuingCA.GetCertificateChain()
}

// GetExpiration returns the time the given certificate expires.
func (c Certificate) GetExpiration() time.Time {
	return c.expiration
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

	x509RootCert, err := DecodePEMCertificate(pemCert)
	if err != nil {
		log.Error().Err(err).Msgf("Error converting certificate from PEM to x509 - CN=%s", rootCertificateName)
	}

	rootCertificate := Certificate{
		commonName: rootCertificateName,
		certChain:  pemCert,
		privateKey: pemKey,
		expiration: x509RootCert.NotAfter,
	}
	return &rootCertificate, nil
}

// NewCertManager creates a new CertManager with the passed CA and CA Private Key
func NewCertManager(ca certificate.Certificater, validity time.Duration) (*CertManager, error) {
	if ca == nil {
		return nil, errNoIssuingCA
	}

	cache := make(map[certificate.CommonName]certificate.Certificater)

	certManager := CertManager{
		// The root certificate signing all newly issued certificates
		ca: ca,

		// Newly issued certificates will be valid for this duration
		validityPeriod: validity,

		// Channel used to inform other components of cert changes (rotation etc.)
		announcements: make(chan interface{}),

		// Certificate cache
		cache: &cache,
	}

	// Setup certificate rotation
	done := make(chan interface{})

	// Instantiating a new certificate rotation mechanism will start a goroutine and return an announcement channel
	// which we use to get notified when a cert has been rotated. From then we pass that onto whoever is listening
	// to the announcement channel of pkg/tresor.
	announcements := rotor.New(checkCertificateExpirationInterval, done, &certManager, &cache)
	go func() {
		for {
			<-announcements
			certManager.announcements <- nil
		}
	}()

	return &certManager, nil
}
