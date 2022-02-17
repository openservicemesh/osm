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
	"github.com/openservicemesh/osm/pkg/errcode"
	"github.com/openservicemesh/osm/pkg/k8s/events"
	"github.com/openservicemesh/osm/pkg/logger"
	"github.com/openservicemesh/osm/pkg/messaging"
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
func NewCertManager(
	vaultAddr,
	token string,
	role string,
	cfg configurator.Configurator,
	serviceCertValidityDuration time.Duration,
	msgBroker *messaging.Broker) (*CertManager, error) {
	c := &CertManager{
		role:                        vaultRole(role),
		cfg:                         cfg,
		serviceCertValidityDuration: serviceCertValidityDuration,
		msgBroker:                   msgBroker,
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

	c.ca = &certificate.Certificate{
		CommonName:   constants.CertificationAuthorityCommonName,
		SerialNumber: serialNumber,
		Expiration:   time.Now().Add(decade),
		CertChain:    issuingCA,
		IssuingCA:    issuingCA,
	}

	// Instantiating a new certificate rotation mechanism will start a goroutine for certificate rotation.
	rotor.New(c).Start(checkCertificateExpirationInterval)

	return c, nil
}

func (cm *CertManager) getIssuingCA(issue func(certificate.CommonName, time.Duration) (*certificate.Certificate, error)) ([]byte, certificate.SerialNumber, error) {
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

func (cm *CertManager) issue(cn certificate.CommonName, validityPeriod time.Duration) (*certificate.Certificate, error) {
	secret, err := cm.client.Logical().Write(getIssueURL(cm.role).String(), getIssuanceData(cn, validityPeriod))
	if err != nil {
		// TODO(#3962): metric might not be scraped before process restart resulting from this error
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrIssuingCert)).
			Msgf("Error issuing new certificate for CN=%s", cn)
		return nil, err
	}

	return newCert(cn, secret, time.Now().Add(validityPeriod)), nil
}

func (cm *CertManager) deleteFromCache(cn certificate.CommonName) {
	cm.cache.Delete(cn)
}

func (cm *CertManager) getFromCache(cn certificate.CommonName) *certificate.Certificate {
	if certificateInterface, exists := cm.cache.Load(cn); exists {
		cert := certificateInterface.(*certificate.Certificate)
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
func (cm *CertManager) IssueCertificate(cn certificate.CommonName, validityPeriod time.Duration) (*certificate.Certificate, error) {
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
func (cm *CertManager) ListCertificates() ([]*certificate.Certificate, error) {
	var certs []*certificate.Certificate
	cm.cache.Range(func(cnInterface interface{}, certInterface interface{}) bool {
		certs = append(certs, certInterface.(*certificate.Certificate))
		return true // continue the iteration
	})
	return certs, nil
}

// GetCertificate returns a certificate given its Common Name (CN)
func (cm *CertManager) GetCertificate(cn certificate.CommonName) (*certificate.Certificate, error) {
	if cert := cm.getFromCache(cn); cert != nil {
		return cert, nil
	}
	return nil, errCertNotFound
}

// GetRootCertificate returns the root certificate.
func (cm *CertManager) GetRootCertificate() (*certificate.Certificate, error) {
	return cm.ca, nil
}

// RotateCertificate implements certificate.Manager and rotates an existing certificate.
func (cm *CertManager) RotateCertificate(cn certificate.CommonName) (*certificate.Certificate, error) {
	start := time.Now()

	oldCert, ok := cm.cache.Load(cn)
	if !ok {
		return nil, errors.Errorf("Old certificate does not exist for CN=%s", cn)
	}

	// We want the validity duration of the CertManager to remain static during the lifetime
	// of the CertManager. This tests to see if this value is set, and if it isn't then it
	// should make the infrequent call to configuration to get this value and cache it for
	// future certificate operations.
	if cm.serviceCertValidityDuration == 0 {
		cm.serviceCertValidityDuration = cm.cfg.GetServiceCertValidityPeriod()
	}
	newCert, err := cm.issue(cn, cm.serviceCertValidityDuration)
	if err != nil {
		return nil, err
	}

	cm.cache.Store(cn, newCert)

	cm.msgBroker.GetCertPubSub().Pub(events.PubSubMessage{
		Kind:   announcements.CertificateRotated,
		NewObj: newCert,
		OldObj: oldCert.(*certificate.Certificate),
	}, announcements.CertificateRotated.String())

	log.Debug().Msgf("Rotated certificate (old SerialNumber=%s) with new SerialNumber=%s took %+v", oldCert.(*certificate.Certificate).GetSerialNumber(), newCert.SerialNumber, time.Since(start))

	return newCert, nil
}

func newCert(cn certificate.CommonName, secret *api.Secret, expiration time.Time) *certificate.Certificate {
	return &certificate.Certificate{
		CommonName:   cn,
		SerialNumber: certificate.SerialNumber(secret.Data[serialNumberField].(string)),
		Expiration:   expiration,
		CertChain:    pem.Certificate(secret.Data[certificateField].(string)),
		PrivateKey:   []byte(secret.Data[privateKeyField].(string)),
		IssuingCA:    pem.RootCertificate(secret.Data[issuingCAField].(string)),
	}
}
