package certificate

import (
	"context"
	"errors"
	"math/rand"
	"sync"
	"time"

	"github.com/openservicemesh/osm/pkg/announcements"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/errcode"
	"github.com/openservicemesh/osm/pkg/k8s/events"
	"github.com/openservicemesh/osm/pkg/logger"
	"github.com/openservicemesh/osm/pkg/messaging"
)

var (
	log = logger.New("certificate")
)

// NewManager creates a new CertificateManager with the passed MRCClient and options
func NewManager(ctx context.Context, mrcClient MRCClient, getServiceCertValidityPeriod func() time.Duration, getIngressCertValidityDuration func() time.Duration, msgBroker *messaging.Broker, checkInterval time.Duration) (*Manager, error) {
	m := &Manager{
		serviceCertValidityDuration: getServiceCertValidityPeriod,
		ingressCertValidityDuration: getIngressCertValidityDuration,
		msgBroker:                   msgBroker,
	}

	err := m.start(ctx, mrcClient)
	if err != nil {
		return nil, err
	}

	m.startRotationTicker(ctx, checkInterval)
	return m, nil
}

func (m *Manager) startRotationTicker(ctx context.Context, checkInterval time.Duration) {
	ticker := time.NewTicker(checkInterval)
	go func() {
		m.checkAndRotate()
		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				return
			case <-ticker.C:
				m.checkAndRotate()
			}
		}
	}()
}

func (m *Manager) start(ctx context.Context, mrcClient MRCClient) error {
	// start a watch and we wait until the manager is initialized so that
	// the caller gets a manager that's ready to be used
	var once sync.Once
	var wg sync.WaitGroup
	mrcEvents, err := mrcClient.Watch(ctx)
	if err != nil {
		return err
	}

	wg.Add(1)

	go func(wg *sync.WaitGroup, once *sync.Once) {
		for {
			select {
			case <-ctx.Done():
				if err := ctx.Err(); err != nil {
					log.Error().Err(err).Msg("context canceled with error. stopping MRC watch...")
					return
				}

				log.Info().Msg("context canceled. stopping MRC watch...")
				return
			case event, open := <-mrcEvents:
				if !open {
					// channel was closed; return
					log.Info().Msg("stopping MRC watch...")
					return
				}

				err = m.handleMRCEvent(mrcClient, event)
				if err != nil {
					log.Error().Err(err).Msgf("error encountered processing MRCEvent")
					continue
				}
			}

			if m.signingIssuer != nil && m.validatingIssuer != nil {
				once.Do(func() {
					wg.Done()
				})
			}
		}
	}(&wg, &once)

	done := make(chan struct{})

	// Wait for WaitGroup to finish and notify select when it does
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-time.After(10 * time.Second):
		// We timed out
		return errors.New("manager initialization timed out. Make sure your MeshRootCertificate(s) are valid")
	case <-done:
	}

	return nil
}

func (m *Manager) handleMRCEvent(mrcClient MRCClient, event MRCEvent) error {
	switch event.Type {
	case MRCEventAdded:
		mrc := event.MRC
		if mrc.Status.State == constants.MRCStateError {
			log.Debug().Msgf("skipping MRC with error state %s", mrc.GetName())
			return nil
		}

		client, ca, err := mrcClient.GetCertIssuerForMRC(mrc)
		if err != nil {
			return err
		}

		c := &issuer{Issuer: client, ID: mrc.Name, CertificateAuthority: ca, TrustDomain: mrc.Spec.TrustDomain}
		switch {
		case mrc.Status.State == constants.MRCStateActive:
			m.mu.Lock()
			m.signingIssuer = c
			m.validatingIssuer = c
			m.mu.Unlock()
		case mrc.Status.State == constants.MRCStateIssuingRollback || mrc.Status.State == constants.MRCStateIssuingRollout:
			m.mu.Lock()
			m.signingIssuer = c
			m.mu.Unlock()
		case mrc.Status.State == constants.MRCStateValidatingRollback || mrc.Status.State == constants.MRCStateValidatingRollout:
			m.mu.Lock()
			m.validatingIssuer = c
			m.mu.Unlock()
		default:
			m.mu.Lock()
			m.signingIssuer = c
			m.validatingIssuer = c
			m.mu.Unlock()
		}
	case MRCEventUpdated:
		// TODO
	}

	return nil
}

// GetTrustDomain returns the trust domain from the configured signingkey issuer.
// Note that the CRD uses a default, so this value will always be set.
func (m *Manager) GetTrustDomain() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.signingIssuer.TrustDomain
}

// shouldRotate determines whether a certificate should be rotated.
func (m *Manager) shouldRotate(c *Certificate) bool {
	// The certificate is going to expire at a timestamp T
	// We want to renew earlier. How much earlier is defined in renewBeforeCertExpires.
	// We add a few seconds noise to the early renew period so that certificates that may have been
	// created at the same time are not renewed at the exact same time.
	intNoise := rand.Intn(noiseSeconds) // #nosec G404
	secondsNoise := time.Duration(intNoise) * time.Second
	renewBefore := RenewBeforeCertExpires + secondsNoise
	if time.Until(c.GetExpiration()) <= renewBefore {
		log.Info().Msgf("Cert %s should be rotated; expires in %+v; renewBefore is %+v",
			c.GetCommonName(),
			time.Until(c.GetExpiration()),
			renewBefore)
		return true
	}

	m.mu.Lock()
	validatingIssuer := m.validatingIssuer
	signingIssuer := m.signingIssuer
	m.mu.Unlock()

	// During root certificate rotation the Issuers will change. If the Manager's Issuers are
	// different than the validating Issuer and signing Issuer IDs in the certificate, the
	// certificate must be reissued with the correct Issuers for the current rotation stage and
	// state. If there is no root certificate rotation in progress, the cert and Manager Issuers
	// will match.
	if c.signingIssuerID != signingIssuer.ID || c.validatingIssuerID != validatingIssuer.ID {
		log.Info().Msgf("Cert %s should be rotated; in progress root certificate rotation",
			c.GetCommonName())
		return true
	}
	return false
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
		opts := []IssueOption{}
		if key == cert.CommonName.String() {
			opts = append(opts, FullCNProvided())
		}

		_, err := m.IssueCertificate(key, cert.certType, opts...)
		if err != nil {
			log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrRotatingCert)).
				Msgf("Error rotating cert SerialNumber=%s", cert.GetSerialNumber())
		}
	}
}

func (m *Manager) getValidityDurationForCertType(ct CertType) time.Duration {
	switch ct {
	case Internal:
		return constants.OSMCertificateValidityPeriod
	case IngressGateway:
		return m.ingressCertValidityDuration()
	case Service:
		return m.serviceCertValidityDuration()
	default:
		log.Debug().Msgf("Unknown certificate type %s provided when getting validity duration", ct)
		return constants.OSMCertificateValidityPeriod
	}
}

// getFromCache returns the certificate with the specified cn from cache if it exists.
// Note: getFromCache might return an expired or invalid certificate.
func (m *Manager) getFromCache(key string) *Certificate {
	certInterface, exists := m.cache.Load(key)
	if !exists {
		return nil
	}
	cert := certInterface.(*Certificate)
	log.Trace().Msgf("Certificate found in cache SerialNumber=%s", cert.GetSerialNumber())
	return cert
}

// IssueCertificate returns a newly issued certificate from the given client
// or an existing valid certificate from the local cache.
func (m *Manager) IssueCertificate(prefix string, ct CertType, opts ...IssueOption) (*Certificate, error) {
	// a singleflight group is used here to ensure that only one issueCertificate is in
	// flight at a time for a given certificate prefix. Helps avoid a race condition if
	// issueCertificate is called multiple times in a row for the same certificate prefix.
	cert, err, _ := m.group.Do(prefix, func() (interface{}, error) {
		return m.issueCertificate(prefix, ct, opts...)
	})
	if err != nil {
		return nil, err
	}
	return cert.(*Certificate), nil
}

func (m *Manager) issueCertificate(prefix string, ct CertType, opts ...IssueOption) (*Certificate, error) {
	var rotate bool
	cert := m.getFromCache(prefix) // Don't call this while holding the lock
	if cert != nil {
		// check if cert needs to be rotated
		rotate = m.shouldRotate(cert)
		if !rotate {
			return cert, nil
		}
	}

	options := &issueOptions{}
	for _, o := range opts {
		o(options)
	}

	m.mu.Lock()
	validatingIssuer := m.validatingIssuer
	signingIssuer := m.signingIssuer
	m.mu.Unlock()

	start := time.Now()

	validityDuration := m.getValidityDurationForCertType(ct)
	newCert, err := signingIssuer.IssueCertificate(options.formatCN(prefix, signingIssuer.TrustDomain), validityDuration)
	if err != nil {
		return nil, err
	}

	// if we have different signing and validating issuers,
	// create the cert's trust context
	if validatingIssuer.ID != signingIssuer.ID {
		newCert = newCert.newMergedWithRoot(validatingIssuer.CertificateAuthority)
	}

	newCert.signingIssuerID = signingIssuer.ID
	newCert.validatingIssuerID = validatingIssuer.ID

	newCert.certType = ct

	m.cache.Store(prefix, newCert)

	log.Trace().Msgf("It took %s to issue certificate with SerialNumber=%s", time.Since(start), newCert.GetSerialNumber())

	if rotate {
		// Certificate was rotated
		m.msgBroker.GetCertPubSub().Pub(events.PubSubMessage{
			Kind:   announcements.CertificateRotated,
			NewObj: newCert,
			OldObj: cert,
		}, announcements.CertificateRotated.String())

		log.Debug().Msgf("Rotated certificate (old SerialNumber=%s) with new SerialNumber=%s", cert.SerialNumber, newCert.SerialNumber)
	}

	return newCert, nil
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
