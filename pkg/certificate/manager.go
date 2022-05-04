package certificate

import (
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"

	"github.com/openservicemesh/osm/pkg/announcements"
	"github.com/openservicemesh/osm/pkg/errcode"
	"github.com/openservicemesh/osm/pkg/k8s/events"
	"github.com/openservicemesh/osm/pkg/messaging"
)

var errCertNotFound = errors.New("certificate not found")

// NewManager creates a new CertManager
func NewManager(
	ca *Certificate,
	certClient client,
	serviceCertValidityDuration time.Duration,
	msgBroker *messaging.Broker) (*Manager, error) {
	m := &Manager{
		clients:                     []client{certClient},
		serviceCertValidityDuration: serviceCertValidityDuration,
		msgBroker:                   msgBroker,
	}
	return m, nil
}

// Start takes an interval to check if the certificate
// needs to be rotated
func (m *Manager) Start(checkInterval time.Duration, certRotation <-chan struct{}) {
	// iterate over the list of certificates
	// when a cert needs to be rotated - call RotateCertificate()
	if certRotation == nil {
		log.Error().Msgf("Cannot start certificate rotation, certRotation is nil")
		return
	}
	ticker := time.NewTicker(checkInterval)
	go func() {
		m.checkAndRotate()
		for {
			select {
			case <-certRotation:
				ticker.Stop()
				return
			case <-ticker.C:
				m.checkAndRotate()
			}
		}
	}()
}

func (m *Manager) checkAndRotate() {
	certs, err := m.ListCertificates()
	if err != nil {
		log.Error().Err(err).Msgf("Error listing all certificates")
	}

	for _, cert := range certs {
		shouldRotate := cert.ShouldRotate()

		word := map[bool]string{true: "will", false: "will not"}[shouldRotate]
		log.Trace().Msgf("Cert %s %s be rotated; expires in %+v; renewBeforeCertExpires is %+v",
			cert.GetCommonName(),
			word,
			time.Until(cert.GetExpiration()),
			RenewBeforeCertExpires)

		if shouldRotate {
			// Remove the certificate from the cache of the certificate manager
			newCert, err := m.RotateCertificate(cert.GetCommonName())
			if err != nil {
				// TODO(#3962): metric might not be scraped before process restart resulting from this error
				log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrRotatingCert)).
					Msgf("Error rotating cert SerialNumber=%s", cert.GetSerialNumber())
				continue
			}
			log.Trace().Msgf("Rotated cert SerialNumber=%s", newCert.GetSerialNumber())
		}
	}
}

func (m *Manager) getFromCache(cn CommonName) *Certificate {
	certInterface, exists := m.cache.Load(cn)
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
func (m *Manager) IssueCertificate(cn CommonName, validityPeriod time.Duration) (*Certificate, error) {
	start := time.Now()

	if cert := m.getFromCache(cn); cert != nil {
		return cert, nil
	}

	// TODO(#4502): determine client(s) to use based on the rotation stage(s) of the client(s)
	cert, err := m.clients[0].IssueCertificate(cn, validityPeriod)
	if err != nil {
		return cert, err
	}

	m.cache.Store(cn, cert)

	log.Trace().Msgf("It took %s to issue certificate with SerialNumber=%s", time.Since(start), cert.GetSerialNumber())

	return cert, nil
}

// ReleaseCertificate is called when a cert will no longer be needed and should be removed from the system.
func (m *Manager) ReleaseCertificate(cn CommonName) {
	log.Trace().Msgf("Releasing certificate %s", cn)
	m.cache.Delete(cn)
}

// GetCertificate returns a certificate given its Common Name (CN)
func (m *Manager) GetCertificate(cn CommonName) (*Certificate, error) {
	if cert := m.getFromCache(cn); cert != nil {
		return cert, nil
	}
	return nil, errCertNotFound
}

// RotateCertificate implements Manager and rotates an existing
func (m *Manager) RotateCertificate(cn CommonName) (*Certificate, error) {
	start := time.Now()

	oldObj, ok := m.cache.Load(cn)
	if !ok {
		return nil, errors.Errorf("Old certificate does not exist for CN=%s", cn)
	}

	oldCert, ok := oldObj.(*Certificate)
	if !ok {
		return nil, errors.Errorf("unexpected type %T for old certificate does not exist for CN=%s", oldCert, cn)
	}

	newCert, err := m.IssueCertificate(cn, m.serviceCertValidityDuration)
	if err != nil {
		return nil, err
	}

	m.cache.Store(cn, newCert)

	m.msgBroker.GetCertPubSub().Pub(events.PubSubMessage{
		Kind:   announcements.CertificateRotated,
		NewObj: newCert,
		OldObj: oldCert,
	}, announcements.CertificateRotated.String())

	log.Debug().Msgf("Rotated certificate (old SerialNumber=%s) with new SerialNumber=%s took %+v", oldCert.SerialNumber, newCert.SerialNumber, time.Since(start))

	return newCert, nil
}

// ListCertificates lists all certificates issued
func (m *Manager) ListCertificates() ([]*Certificate, error) {
	var certs []*Certificate
	m.cache.Range(func(_ interface{}, certInterface interface{}) bool {
		certs = append(certs, certInterface.(*Certificate))
		return true // continue the iteration
	})
	return certs, nil
}

// GetRootCertificate returns the root
func (m *Manager) GetRootCertificate() *Certificate {
	// TODO(#4502): determine client(s) to use based on the rotation stage(s) of the client(s)
	return m.clients[0].GetRootCertificate()
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
