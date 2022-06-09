package certificate

import (
	"time"

	"github.com/rs/zerolog/log"

	"github.com/openservicemesh/osm/pkg/announcements"
	"github.com/openservicemesh/osm/pkg/errcode"
	"github.com/openservicemesh/osm/pkg/k8s/events"
	"github.com/openservicemesh/osm/pkg/messaging"
)

// NewManager creates a new CertManager with the passed CA and CA Private Key
func NewManager(mrcClient MRCClient, serviceCertValidityDuration time.Duration, msgBroker *messaging.Broker) (*Manager, error) {
	// TODO(#4502): transition this call to a watch function that knows how to handle multiple MRC and can react to changes.
	mrcs, err := mrcClient.List()
	if err != nil {
		return nil, err
	}

	client, ca, clientID, err := mrcClient.GetCertIssuerForMRC(mrcs[0])
	if err != nil {
		return nil, err
	}

	c := &issuer{Issuer: client, ID: clientID, CertificateAuthority: ca, TrustDomain: mrcs[0].Spec.TrustDomain}

	m := &Manager{
		// The signingIssuer is responsible for signing all newly issued certificates
		// The validatingIssuer is the issuer that issued existing certificates.
		// its underlying cert is still in the validating trust store
		signingIssuer:               c,
		validatingIssuer:            c,
		serviceCertValidityDuration: serviceCertValidityDuration,
		msgBroker:                   msgBroker,
	}
	return m, nil
}

// Start takes an interval to check if the certificate
// needs to be rotated
func (m *Manager) Start(checkInterval time.Duration, stop <-chan struct{}) {
	ticker := time.NewTicker(checkInterval)
	go func() {
		m.checkAndRotate()
		for {
			select {
			case <-stop:
				ticker.Stop()
				return
			case <-ticker.C:
				m.checkAndRotate()
			}
		}
	}()
}

// GetTrustDomain returns the trust domain from the configured signingkey issuer.
// Note that the CRD uses a default, so this value will always be set.
func (m *Manager) GetTrustDomain() string {
	// TODO(4754): implement
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.signingIssuer.TrustDomain
}

func (m *Manager) checkAndRotate() {
	// NOTE: checkAndRotate can reintroduce a certificate that has been released, thereby creating an unbounded cache.
	// A certificate can also have been rotated already, leaving the list of issued certs stale, and we re-rotate.
	// the latter is not a bug, but a source of inefficiency.

	certs := map[string]*Certificate{}
	m.cache.Range(func(keyIface interface{}, certInterface interface{}) bool {
		key := keyIface.(string)
		certs[key] = certInterface.(*Certificate)
		return true // continue the iteration
	})
	for key, cert := range certs {
		shouldRotate := cert.ShouldRotate()

		word := map[bool]string{true: "will", false: "will not"}[shouldRotate]
		log.Trace().Msgf("Cert %s %s be rotated; expires in %+v; renewBeforeCertExpires is %+v",
			cert.GetCommonName(),
			word,
			time.Until(cert.GetExpiration()),
			RenewBeforeCertExpires)

		if shouldRotate {
			opts := []IssueOption{WithValidityPeriod(m.serviceCertValidityDuration)}
			// if the key is equal to the common name, then it was issued with FullCNProvided(). This will prevent
			// an additional trust domain from being appended. We don't do this in every case, in case the trust domain
			// has changed since the last issue.
			if key == cert.CommonName.String() {
				opts = append(opts, FullCNProvided())
			}
			newCert, err := m.IssueCertificate(key, opts...)
			if err != nil {
				// TODO(#3962): metric might not be scraped before process restart resulting from this error
				log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrRotatingCert)).
					Msgf("Error rotating cert SerialNumber=%s", cert.GetSerialNumber())
				continue
			}

			m.msgBroker.GetCertPubSub().Pub(events.PubSubMessage{
				Kind:   announcements.CertificateRotated,
				NewObj: newCert,
				OldObj: cert,
			}, announcements.CertificateRotated.String())

			log.Debug().Msgf("Rotated certificate (old SerialNumber=%s) with new SerialNumber=%s", cert.SerialNumber, newCert.SerialNumber)
		}
	}
}

func (m *Manager) getFromCache(key string) *Certificate {
	certInterface, exists := m.cache.Load(key)
	if !exists {
		return nil
	}
	cert := certInterface.(*Certificate)
	log.Trace().Msgf("Certificate found in cache SerialNumber=%s", cert.GetSerialNumber())
	if cert.ShouldRotate() {
		log.Trace().Msgf("Certificate found in cache but has expired SerialNumber=%s", cert.GetSerialNumber())
		return nil
	}
	return cert
}

// IssueCertificate implements Manager and returns a newly issued certificate from the given client.
func (m *Manager) IssueCertificate(prefix string, opts ...IssueOption) (*Certificate, error) {
	var err error
	cert := m.getFromCache(prefix) // Don't call this while holding the lock

	options := defaultOptions(m.serviceCertValidityDuration)

	for _, o := range opts {
		o(options)
	}

	m.mu.RLock()
	validatingIssuer := m.validatingIssuer
	signingIssuer := m.signingIssuer
	m.mu.RUnlock()

	start := time.Now()
	if cert == nil || cert.signingIssuerID != signingIssuer.ID || cert.validatingIssuerID != validatingIssuer.ID {
		cert, err = signingIssuer.IssueCertificate(options.formatCN(prefix, signingIssuer.TrustDomain), options.validityPeriod)
		if err != nil {
			return nil, err
		}

		// if we have different signing and validating issuers,
		// create the cert's trust context
		if validatingIssuer.ID != signingIssuer.ID {
			cert = cert.newMergedWithRoot(validatingIssuer.CertificateAuthority)
		}

		cert.signingIssuerID = signingIssuer.ID
		cert.validatingIssuerID = validatingIssuer.ID
	}

	m.cache.Store(prefix, cert)

	log.Trace().Msgf("It took %s to issue certificate with SerialNumber=%s", time.Since(start), cert.GetSerialNumber())

	return cert, nil
}

// ReleaseCertificate is called when a cert will no longer be needed and should be removed from the system.
func (m *Manager) ReleaseCertificate(key string) {
	log.Trace().Msgf("Releasing certificate %s", key)
	m.cache.Delete(key)
}

// ListIssuedCertificates implements CertificateDebugger interface and returns the list of issued certificates.
func (m *Manager) ListIssuedCertificates() []*Certificate {
	var certs []*Certificate
	m.cache.Range(func(cnInterface interface{}, certInterface interface{}) bool {
		certs = append(certs, certInterface.(*Certificate))
		return true // continue the iteration
	})
	return certs
}
