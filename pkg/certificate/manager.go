package certificate

import (
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"

	"github.com/openservicemesh/osm/pkg/announcements"
	"github.com/openservicemesh/osm/pkg/k8s/events"
	"github.com/openservicemesh/osm/pkg/messaging"
)

var (
	errNoIssuingCA  = errors.New("no issuing CA")
	errCertNotFound = errors.New("certificate not found")
)

// NewManager creates a new CertManager with the passed CA and CA Private Key
func NewManager(
	ca *Certificate,
	client client,
	serviceCertValidityDuration time.Duration,
	msgBroker *messaging.Broker) (*manager, error) { //nolint:revive // unexported-return
	if ca == nil {
		return nil, errNoIssuingCA
	}

	m := &manager{
		// The root certificate signing all newly issued certificates
		ca:                          ca,
		client:                      client,
		serviceCertValidityDuration: serviceCertValidityDuration,
		msgBroker:                   msgBroker,
	}

	// TODO(#4533) start the cert rotation here.

	return m, nil
}

func (m *manager) getFromCache(cn CommonName) *Certificate {
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
func (m *manager) IssueCertificate(cn CommonName, validityPeriod time.Duration) (*Certificate, error) {
	start := time.Now()

	if cert := m.getFromCache(cn); cert != nil {
		return cert, nil
	}

	cert, err := m.client.IssueCertificate(cn, validityPeriod)
	if err != nil {
		return cert, err
	}

	m.cache.Store(cn, cert)

	log.Trace().Msgf("It took %s to issue certificate with SerialNumber=%s", time.Since(start), cert.GetSerialNumber())

	return cert, nil
}

// ReleaseCertificate is called when a cert will no longer be needed and should be removed from the system.
func (m *manager) ReleaseCertificate(cn CommonName) {
	log.Trace().Msgf("Releasing certificate %s", cn)
	m.cache.Delete(cn)
}

// GetCertificate returns a certificate given its Common Name (CN)
func (m *manager) GetCertificate(cn CommonName) (*Certificate, error) {
	if cert := m.getFromCache(cn); cert != nil {
		return cert, nil
	}
	return nil, errCertNotFound
}

// RotateCertificate implements Manager and rotates an existing
func (m *manager) RotateCertificate(cn CommonName) (*Certificate, error) {
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
func (m *manager) ListCertificates() ([]*Certificate, error) {
	var certs []*Certificate
	m.cache.Range(func(_ interface{}, certInterface interface{}) bool {
		certs = append(certs, certInterface.(*Certificate))
		return true // continue the iteration
	})
	return certs, nil
}

// GetRootCertificate returns the root
// TODO(#4533): remove the error from return value if not needed.
func (m *manager) GetRootCertificate() (*Certificate, error) {
	return m.ca, nil
}

// ListIssuedCertificates implements CertificateDebugger interface and returns the list of issued certificates.
func (m *manager) ListIssuedCertificates() []*Certificate {
	var certs []*Certificate
	m.cache.Range(func(cnInterface interface{}, certInterface interface{}) bool {
		certs = append(certs, certInterface.(*Certificate))
		return true // continue the iteration
	})
	return certs
}
