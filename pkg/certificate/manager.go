package certificate

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/cskr/pubsub"

	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/errcode"
	"github.com/openservicemesh/osm/pkg/logger"
)

var (
	log = logger.New("certificate")
)

// NewManager creates a new CertificateManager with the passed MRCClient and options
func NewManager(ctx context.Context, mrcClient MRCClient, getServiceCertValidityPeriod func() time.Duration, getIngressCertValidityDuration func() time.Duration, checkInterval time.Duration) (*Manager, error) {
	m := &Manager{
		serviceCertValidityDuration: getServiceCertValidityPeriod,
		ingressCertValidityDuration: getIngressCertValidityDuration,
		pubsub:                      pubsub.New(1),
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
		mrc := event.NewMRC

		// ignore add event if MRC has no state
		if mrc.Status.State == "" {
			log.Debug().Msgf("received MRC add event for MRC %s. Ignoring because MRC has no state", mrc.GetName())
			return nil
		}
		// ignore add event if MRC state is error
		if mrc.Status.State == constants.MRCStateError {
			log.Debug().Msgf("received MRC add event for MRC %s. Skipping MRC with error state", mrc.GetName())
			return nil
		}

		client, ca, err := mrcClient.GetCertIssuerForMRC(mrc)
		if err != nil {
			return err
		}

		c := &issuer{Issuer: client, ID: mrc.Name, CertificateAuthority: ca, TrustDomain: mrc.Spec.TrustDomain}
		switch mrc.Status.State {
		case constants.MRCStateActive:
			m.mu.Lock()
			m.signingIssuer = c
			m.validatingIssuer = c
			m.mu.Unlock()
		case constants.MRCStateIssuingRollout:
			m.mu.Lock()
			m.signingIssuer = c
			m.mu.Unlock()
		case constants.MRCStateIssuingRollback, constants.MRCStateValidatingRollout:
			m.mu.Lock()
			m.validatingIssuer = c
			m.mu.Unlock()
		case constants.MRCStateValidatingRollback:
			// do nothing, the issuer of active MRC should be used
		default:
			m.mu.Lock()
			m.signingIssuer = c
			m.validatingIssuer = c
			m.mu.Unlock()
		}
	case MRCEventUpdated:
		newMRC := event.NewMRC
		oldMRC := event.OldMRC
		if newMRC.Status.State != oldMRC.Status.State {
			// TODO(jaellio): Do I need to validate status updates here to determine if they are allowed?
			return m.handleMRCStatusUpdate(mrcClient, event)
		}
		log.Debug().Msgf("Did not receive an MRC %s update requiring a rotation", newMRC.GetName())
		return nil
	}

	return nil
}

func (m *Manager) handleMRCStatusUpdate(mrcClient MRCClient, event MRCEvent) error {
	newMRC := event.NewMRC
	oldMRC := event.OldMRC
	log.Debug().Msgf("handling MRC status update for MRC %s/%s from %s state to %s state",
		newMRC.GetNamespace(), newMRC.GetName(), oldMRC.Status.State, newMRC.Status.State)

	switch newMRC.Status.State {
	case constants.MRCStateValidatingRollout:
		client, ca, err := mrcClient.GetCertIssuerForMRC(newMRC)
		if err != nil {
			return err
		}

		c := &issuer{Issuer: client, ID: newMRC.Name, CertificateAuthority: ca, TrustDomain: newMRC.Spec.TrustDomain}

		m.mu.Lock()
		m.validatingIssuer = c
		m.mu.Unlock()
		log.Debug().Msgf("successfully updated validating issuer to the provider specified in MRC %s/%s",
			newMRC.GetNamespace(), newMRC.GetName())
	case constants.MRCStateIssuingRollout:
		// TODO(jaellio): Does accessing the ID need to be locked? Should this lock be help for the entire time we process the event?
		m.mu.Lock()
		validatingIssuerID := m.validatingIssuer.ID
		signingIssuerID := m.signingIssuer.ID
		m.mu.Unlock()

		if newMRC.GetName() != validatingIssuerID {
			msg := fmt.Sprintf("expected MRC %s/%s with updated status %s to be the current validatingIssuer %s",
				newMRC.GetNamespace(), newMRC.GetName(), constants.MRCStateIssuingRollout, validatingIssuerID)
			// TODO(jaellio): set the MRC error state?
			log.Error().Msg(msg)
			return errors.New(msg)
		}

		m.mu.Lock()
		tempIssuer := m.signingIssuer
		m.signingIssuer = m.validatingIssuer
		m.validatingIssuer = tempIssuer
		m.mu.Unlock()
		log.Debug().Msgf("successfully updated signing issuer to the provider specified in MRC %s/%s. The validating issuer was updated to be signing issuer specified in MRC %s",
			newMRC.GetNamespace(), newMRC.GetName(), signingIssuerID)
	case constants.MRCStateActive:
		// TODO(jaellio): Does accessing the ID need to be locked? Should this lock be help for the entire time we process the event?
		m.mu.Lock()
		signingIssuerID := m.signingIssuer.ID
		m.mu.Unlock()

		if newMRC.GetName() != signingIssuerID {
			msg := fmt.Sprintf("expected MRC %s/%s with updated status %s to be the current signingIssuer %s",
				newMRC.GetNamespace(), newMRC.GetName(), constants.MRCStateActive, signingIssuerID)
			log.Error().Msg(msg)
			return errors.New(msg)
		}

		m.mu.Lock()
		m.validatingIssuer = m.signingIssuer
		m.mu.Unlock()
		log.Debug().Msgf("successfully updated validating issuer to the provider specified in MRC %s/%s",
			newMRC.GetNamespace(), newMRC.GetName())
	case constants.MRCStateIssuingRollback, constants.MRCStateValidatingRollback:
		log.Info().Msgf("no operation required on %s status update to MRC %s", newMRC.Status.State, newMRC.Name)
	case constants.MRCStateInactive:
		// TODO(#4896): clean up inactive MRCs
	case constants.MRCStateError:
		// TODO(#4893): handle MRC error state and potentially perform a rollback
	}
	return nil
}

// GetTrustDomain returns the trust domain from the configured signing key issuer.
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
		m.pubsub.Pub(newCert, prefix)

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

// SubscribeRotations returns a channel that outputs every certificate that is rotated by the manager.
// The caller must call the returned method to close the channel.
// WARNING: you cannot call wait on the returned channel on the same go routine you are issuing a certificate on.
func (m *Manager) SubscribeRotations(key string) (chan interface{}, func()) {
	ch := m.pubsub.Sub(key)
	return ch, func() {
		go m.pubsub.Unsub(ch)
		// must empty the channel to prevent deadlock
		// https://github.com/openservicemesh/osm/issues/4847
		for range ch {
		}
	}
}
