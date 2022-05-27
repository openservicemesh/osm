package certificate

import (
	"math/rand"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/openservicemesh/osm/pkg/announcements"
	"github.com/openservicemesh/osm/pkg/errcode"
	"github.com/openservicemesh/osm/pkg/k8s/events"
	"github.com/openservicemesh/osm/pkg/messaging"
)

// NewManager creates a new CertManager with the passed CA and CA Private Key
func NewManager(mrcClient MRCClient, getServiceCertValidityPeriod func() time.Duration, msgBroker *messaging.Broker) (*Manager, error) {
	// TODO(#4502): transition this call to a watch function that knows how to handle multiple MRC and can react to changes.
	mrcs, err := mrcClient.List()
	if err != nil {
		return nil, err
	}

	client, clientID, err := mrcClient.GetCertIssuerForMRC(mrcs[0])
	if err != nil {
		return nil, err
	}

	c := &issuer{Issuer: client, ID: clientID}

	m := &Manager{
		// The root certificate signing all newly issued certificates
		keyIssuer:                   c,
		pubIssuer:                   c,
		serviceCertValidityDuration: getServiceCertValidityPeriod,
		msgBroker:                   msgBroker,
	}
	return m, nil
}

// Start takes an interval to check if the certificate
// needs to be rotated
func (m *Manager) Start(checkInterval time.Duration, stop <-chan struct{}) {
	ticker := time.NewTicker(checkInterval)
	go func() {
		m.checkAndRotateServiceCerts()
		for {
			select {
			case <-stop:
				ticker.Stop()
				return
			case <-ticker.C:
				m.checkAndRotateServiceCerts()
			}
		}
	}()
}

func (m *Manager) checkAndRotateServiceCerts() {
	// NOTE: checkAndRotate can reintroduce a certificate that has been released, thereby creating an unbounded cache.
	// A certificate can also have been rotated already, leaving the list of issued certs stale, and we re-rotate.
	// the latter is not a bug, but a source of inefficiency.
	for _, cert := range m.ListIssuedCertificates() {
		if cert.GetCertificateType() != Service {
			log.Debug().Msgf("Skipping check and rotate of cert %s of type %s since it is not of type Service",
				cert.GetCommonName(),
				cert.GetCertificateType())
			continue
		}

		newCert, rotated, err := m.IssueCertificate(cert.GetCommonName(), m.serviceCertValidityDuration(), Service)
		if err != nil {
			// TODO(#3962): metric might not be scraped before process restart resulting from this error
			log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrRotatingCert)).
				Msgf("Error getting or issuing cert SerialNumber=%s", cert.GetSerialNumber())
			continue
		}

		word := map[bool]string{true: "was", false: "was not"}[rotated]
		log.Trace().Msgf("Cert %s %s rotated; expires in %+v; renewBeforeCertExpires is %+v",
			cert.GetCommonName(),
			word,
			time.Until(cert.GetExpiration()),
			RenewBeforeCertExpires)

		if rotated {
			m.msgBroker.GetCertPubSub().Pub(events.PubSubMessage{
				Kind:   announcements.CertificateRotated,
				NewObj: newCert,
				OldObj: cert,
			}, announcements.CertificateRotated.String())

			log.Debug().Msgf("Rotated certificate (old SerialNumber=%s) with new SerialNumber=%s", cert.SerialNumber, newCert.SerialNumber)
		}
	}
}

// getFromCache returns the certificate from the cache obtained using the certificate's CN.
// If the certificate should be rotated due to expiration or an in progress root certificate rotation
// getFromCache will return nil even if the certificate is present in the cache.
func (m *Manager) getFromCache(cn CommonName) *Certificate {
	certInterface, exists := m.cache.Load(cn)
	if !exists {
		return nil
	}
	cert := certInterface.(*Certificate)
	log.Trace().Msgf("Certificate found in cache SerialNumber=%s", cert.GetSerialNumber())
	if m.ShouldRotate(cert) {
		log.Trace().Msgf("Certificate found in cache but has expired SerialNumber=%s", cert.GetSerialNumber())
		return nil
	}
	return cert
}

// ShouldRotate determines whether a certificate should be rotated.
func (m *Manager) ShouldRotate(cert *Certificate) bool {
	// The certificate is going to expire at a timestamp T
	// We want to renew earlier. How much earlier is defined in renewBeforeCertExpires.
	// We add a few seconds noise to the early renew period so that certificates that may have been
	// created at the same time are not renewed at the exact same time.

	intNoise := rand.Intn(noiseSeconds) // #nosec G404
	secondsNoise := time.Duration(intNoise) * time.Second
	if time.Until(cert.GetExpiration()) <= (RenewBeforeCertExpires + secondsNoise) {
		return true
	}

	m.mu.RLock()
	pubIssuer := m.pubIssuer
	keyIssuer := m.keyIssuer
	m.mu.RUnlock()

	// During root certificate rotation the Issuers will change. If the Manager's Issuers are
	// different than the key Issuer and cert Issuer IDs in the certificate, the certificate must
	// be reissued with the correct Issuers for the current rotation stage and state.
	// If there is no root certificate rotation in progress, the cert and Manager Issuers will
	// match.
	return cert.keyIssuerID != keyIssuer.ID || cert.pubIssuerID != pubIssuer.ID
}

// IssueCertificate implements Manager and returns a newly issued certificate from the given client.
func (m *Manager) IssueCertificate(cn CommonName, validityPeriod time.Duration, ct CertificateType) (cert *Certificate, rotated bool, err error) {
	cert = m.getFromCache(cn) // Don't call this while holding the lock

	if cert != nil {
		return
	}

	start := time.Now()
	m.mu.RLock()
	pubIssuer := m.pubIssuer
	keyIssuer := m.keyIssuer
	m.mu.RUnlock()

	cert, err = keyIssuer.IssueCertificate(cn, validityPeriod)
	if err != nil {
		return
	}
	var pubCert *Certificate
	if pubIssuer.ID != keyIssuer.ID {
		pubCert, err = pubIssuer.IssueCertificate(cn, validityPeriod)
		if err != nil {
			return
		}

		cert = cert.newMergedWithRoot(pubCert.GetIssuingCA())
	}

	cert.keyIssuerID = keyIssuer.ID
	cert.pubIssuerID = pubIssuer.ID

	cert.CertificateType = ct

	rotated = true

	m.cache.Store(cn, cert)

	log.Trace().Msgf("It took %s to issue certificate with SerialNumber=%s", time.Since(start), cert.GetSerialNumber())

	return
}

// ReleaseCertificate is called when a cert will no longer be needed and should be removed from the system.
func (m *Manager) ReleaseCertificate(cn CommonName) {
	log.Trace().Msgf("Releasing certificate %s", cn)
	m.cache.Delete(cn)
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
