package vault

import (
	"time"

	"github.com/hashicorp/vault/api"

	"github.com/open-service-mesh/osm/pkg/certificate"
	"github.com/open-service-mesh/osm/pkg/certificate/providers/tresor/pem"
	"github.com/open-service-mesh/osm/pkg/certificate/rotor"
	"github.com/open-service-mesh/osm/pkg/constants"
	"github.com/open-service-mesh/osm/pkg/logger"
)

var log = logger.New("vault")

const (
	certificateField = "certificate"
	privateKeyField  = "private_key"
	issuingCAField   = "issuing_ca"

	checkCertificateExpirationInterval = 5 * time.Second
)

// NewCertManager implements certificate.Manager and wraps a Hashi Vault with methods to allow easy certificate issuance.
func NewCertManager(vaultAddr, token string, validityPeriod time.Duration, vaultRole string) (*CertManager, error) {
	cache := make(map[certificate.CommonName]certificate.Certificater)
	c := &CertManager{
		validityPeriod: validityPeriod,
		announcements:  make(chan interface{}),
		cache:          &cache,
		vaultRole:      vaultRole,
	}
	config := api.DefaultConfig()
	config.Address = vaultAddr

	var err error
	if c.client, err = api.NewClient(config); err != nil {
		log.Fatal().Err(err).Msgf("Error creating Vault CertManager without TLS at %s", vaultAddr)
		return nil, err
	}

	log.Info().Msgf("Created Vault CertManager, with vaultRole=%q at %v", vaultRole, vaultAddr)

	c.client.SetToken(token)

	someCert, err := c.issue("localhost", nil)
	if err != nil {
		return nil, err
	}

	c.ca = &Certificate{
		commonName: constants.CertificationAuthorityCommonName,
		expiration: time.Now().Add(8765 * time.Hour), // a decade
		certChain:  someCert.GetIssuingCA(),
		issuingCA:  someCert.GetIssuingCA(),
	}

	// Setup certificate rotation
	done := make(chan interface{})

	// Instantiating a new certificate rotation mechanism will start a goroutine and return an announcement channel
	// which we use to get notified when a cert has been rotated. From then we pass that onto whoever is listening
	// to the announcement channel of pkg/tresor.
	announcements := rotor.New(checkCertificateExpirationInterval, done, c, &cache)
	go func() {
		for {
			<-announcements
			c.announcements <- nil
		}
	}()

	return c, nil
}

func (cm *CertManager) issue(cn certificate.CommonName, validityPeriod *time.Duration) (certificate.Certificater, error) {
	secret, err := cm.client.Logical().Write(getIssueURL(cm.vaultRole), getIssuanceData(cn, cm.validityPeriod))
	if err != nil {
		log.Error().Err(err).Msgf("Error issuing new certificate for CN=%s", cn)
		return nil, err
	}

	if validityPeriod == nil {
		validityPeriod = &cm.validityPeriod
	}

	return newCert(cn, secret, time.Now().Add(*validityPeriod)), nil
}

func (cm *CertManager) getFromCache(cn certificate.CommonName) certificate.Certificater {
	cm.cacheLock.Lock()
	defer cm.cacheLock.Unlock()
	if cert, exists := (*cm.cache)[cn]; exists {
		log.Trace().Msgf("Found in cache certificate with CN=%s", cn)
		return cert
	}
	return nil
}

// IssueCertificate issues a certificate by leveraging the Hashi Vault CertManager.
func (cm *CertManager) IssueCertificate(cn certificate.CommonName, validityPeriod *time.Duration) (certificate.Certificater, error) {
	log.Info().Msgf("Issuing new certificate for CN=%s", cn)

	start := time.Now()

	if cert := cm.getFromCache(cn); cert != nil {
		return cert, nil
	}

	cert, err := cm.issue(cn, validityPeriod)
	if err != nil {
		return cert, err
	}

	cm.cacheLock.Lock()
	(*cm.cache)[cn] = cert
	cm.cacheLock.Unlock()

	log.Info().Msgf("Issuing new certificate for CN=%s took %+v", cn, time.Since(start))

	return cert, nil
}

// GetRootCertificate returns the root certificate.
func (cm *CertManager) GetRootCertificate() (certificate.Certificater, error) {
	return cm.ca, nil
}

// GetAnnouncementsChannel returns a channel used by the Hashi Vault instance to signal when a certificate has been changed.
func (cm *CertManager) GetAnnouncementsChannel() <-chan interface{} {
	return cm.announcements
}

// RotateCertificate implements certificate.Manager and rotates an existing certificate.
func (cm *CertManager) RotateCertificate(cn certificate.CommonName) (certificate.Certificater, error) {
	log.Info().Msgf("Rotating certificate for CN=%s", cn)

	start := time.Now()

	cert, err := cm.issue(cn, &cm.validityPeriod)
	if err != nil {
		return cert, err
	}

	cm.cacheLock.Lock()
	(*cm.cache)[cn] = cert
	cm.cacheLock.Unlock()

	log.Info().Msgf("Rotating certificate CN=%s took %+v", cn, time.Since(start))

	return cert, nil
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

	// Certificate authority signing this certificate.
	issuingCA pem.RootCertificate
}

// GetCommonName returns the common name of the given certificate.
func (c Certificate) GetCommonName() certificate.CommonName {
	return c.commonName
}

// GetCertificateChain returns the PEM encoded certificate.
func (c Certificate) GetCertificateChain() []byte {
	return c.certChain
}

// GetPrivateKey returns the PEM encoded private key of the given certificate.
func (c Certificate) GetPrivateKey() []byte {
	return c.privateKey
}

// GetIssuingCA returns the root certificate signing the given cert.
func (c Certificate) GetIssuingCA() []byte {
	return c.issuingCA
}

// GetExpiration implements certificate.Certificater and returns the time the given certificate expires.
func (c Certificate) GetExpiration() time.Time {
	return c.expiration
}

func newCert(cn certificate.CommonName, secret *api.Secret, expiration time.Time) *Certificate {
	return &Certificate{
		commonName: cn,
		expiration: expiration,
		certChain:  pem.Certificate(secret.Data[certificateField].(string)),
		privateKey: []byte(secret.Data[privateKeyField].(string)),
		issuingCA:  pem.RootCertificate(secret.Data[issuingCAField].(string)),
	}
}
