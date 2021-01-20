package vault

import (
	"time"

	"github.com/hashicorp/vault/api"
	"github.com/pkg/errors"

	"github.com/openservicemesh/osm/pkg/announcements"
	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/certificate/pem"
	"github.com/openservicemesh/osm/pkg/certificate/rotor"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/logger"
)

var log = logger.New("vault")

const (
	// The string value of the JSON key containing the certificate's Serial Number.
	// See: https://www.vaultproject.io/api-docs/secret/pki#sample-response-8
	serialNumberField = "serial_number"
	certificateField  = "certificate"
	privateKeyField   = "private_key"
	issuingCAField    = "issuing_ca"
	commonNameField   = "common_name"
	ttlField          = "ttl"

	checkCertificateExpirationInterval = 5 * time.Second
	decade                             = 8765 * time.Hour
)

// NewCertManager implements certificate.Manager and wraps a Hashi Vault with methods to allow easy certificate issuance.
func NewCertManager(vaultAddr, token string, role string, cfg configurator.Configurator) (*CertManager, error) {
	c := &CertManager{
		announcements: make(chan announcements.Announcement),
		role:          vaultRole(role),
		cfg:           cfg,
	}
	config := api.DefaultConfig()
	config.Address = vaultAddr

	var err error
	if c.client, err = api.NewClient(config); err != nil {
		return nil, errors.Errorf("Error creating Vault CertManager without TLS at %s", vaultAddr)
	}

	log.Info().Msgf("Created Vault CertManager, with role=%q at %v", role, vaultAddr)

	c.client.SetToken(token)

	issuingCA, serialNumber, err := c.getIssuingCA(c.issue)
	if err != nil {
		return nil, err
	}

	c.ca = &Certificate{
		commonName:   constants.CertificationAuthorityCommonName,
		serialNumber: serialNumber,
		expiration:   time.Now().Add(decade),
		certChain:    issuingCA,
		issuingCA:    issuingCA,
	}

	// Instantiating a new certificate rotation mechanism will start a goroutine for certificate rotation.
	rotor.New(c).Start(checkCertificateExpirationInterval)

	return c, nil
}

func (cm *CertManager) getIssuingCA(issue func(certificate.CommonName, time.Duration) (certificate.Certificater, error)) ([]byte, certificate.SerialNumber, error) {
	// Create a temp certificate to determine the public part of the issuing CA
	cert, err := issue("localhost", decade)
	if err != nil {
		return nil, "", err
	}

	issuingCA := cert.GetIssuingCA()

	// We are not going to need this certificate - remove it
	cm.ReleaseCertificate(cert.GetCommonName())

	return issuingCA, cert.GetSerialNumber(), err
}

func (cm *CertManager) issue(cn certificate.CommonName, validityPeriod time.Duration) (certificate.Certificater, error) {
	secret, err := cm.client.Logical().Write(getIssueURL(cm.role).String(), getIssuanceData(cn, validityPeriod))
	if err != nil {
		log.Error().Err(err).Msgf("Error issuing new certificate for CN=%s", cn)
		return nil, err
	}

	return newCert(cn, secret, time.Now().Add(validityPeriod)), nil
}

func (cm *CertManager) deleteFromCache(cn certificate.CommonName) {
	cm.cache.Delete(cn)
}

func (cm *CertManager) getFromCache(cn certificate.CommonName) certificate.Certificater {
	if certificateInterface, exists := cm.cache.Load(cn); exists {
		cert := certificateInterface.(certificate.Certificater)
		log.Trace().Msgf("Certificate found in cache SerialNumber=%s", cert.GetSerialNumber())
		if rotor.ShouldRotate(cert) {
			log.Trace().Msgf("Certificate found in cache but has expired SerialNumber=%s", cert.GetSerialNumber())
			return nil
		}
		return cert
	}
	return nil
}

// IssueCertificate issues a certificate by leveraging the Hashi Vault CertManager.
func (cm *CertManager) IssueCertificate(cn certificate.CommonName, validityPeriod time.Duration) (certificate.Certificater, error) {
	start := time.Now()

	if cert := cm.getFromCache(cn); cert != nil {
		return cert, nil
	}

	cert, err := cm.issue(cn, validityPeriod)
	if err != nil {
		return cert, err
	}

	cm.cache.Store(cn, cert)

	log.Trace().Msgf("Issued new certificate with SerialNumber=%s took %+v", cert.GetSerialNumber(), time.Since(start))

	return cert, nil
}

// ReleaseCertificate is called when a cert will no longer be needed and should be removed from the system.
func (cm *CertManager) ReleaseCertificate(cn certificate.CommonName) {
	// TODO(draychev): implement Hashicorp Vault delete-cert API here: https://github.com/openservicemesh/osm/issues/2068
	cm.deleteFromCache(cn)
}

// ListCertificates lists all certificates issued
func (cm *CertManager) ListCertificates() ([]certificate.Certificater, error) {
	var certs []certificate.Certificater
	cm.cache.Range(func(cnInterface interface{}, certInterface interface{}) bool {
		certs = append(certs, certInterface.(certificate.Certificater))
		return true // continue the iteration
	})
	return certs, nil
}

// GetCertificate returns a certificate given its Common Name (CN)
func (cm *CertManager) GetCertificate(cn certificate.CommonName) (certificate.Certificater, error) {
	if cert := cm.getFromCache(cn); cert != nil {
		return cert, nil
	}
	return nil, errCertNotFound
}

// GetRootCertificate returns the root certificate.
func (cm *CertManager) GetRootCertificate() (certificate.Certificater, error) {
	return cm.ca, nil
}

// GetAnnouncementsChannel returns a channel used by the Hashi Vault instance to signal when a certificate has been changed.
func (cm *CertManager) GetAnnouncementsChannel() <-chan announcements.Announcement {
	return cm.announcements
}

// RotateCertificate implements certificate.Manager and rotates an existing certificate.
func (cm *CertManager) RotateCertificate(cn certificate.CommonName) (certificate.Certificater, error) {
	start := time.Now()

	cert, err := cm.issue(cn, cm.cfg.GetServiceCertValidityPeriod())
	if err != nil {
		return cert, err
	}

	cm.cache.Store(cn, cert)
	cm.announcements <- announcements.Announcement{}

	log.Trace().Msgf("Rotated certificate with new SerialNumber=%s took %+v", cert.GetSerialNumber(), time.Since(start))

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

	// serialNumber is the serial_number value in the Data field assigned to the Certificate Hashicorp Vault issued
	serialNumber certificate.SerialNumber
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
		commonName:   cn,
		serialNumber: certificate.SerialNumber(secret.Data[serialNumberField].(string)),
		expiration:   expiration,
		certChain:    pem.Certificate(secret.Data[certificateField].(string)),
		privateKey:   []byte(secret.Data[privateKeyField].(string)),
		issuingCA:    pem.RootCertificate(secret.Data[issuingCAField].(string)),
	}
}

// GetSerialNumber returns the serial number of the given certificate.
func (c Certificate) GetSerialNumber() certificate.SerialNumber {
	return c.serialNumber
}
