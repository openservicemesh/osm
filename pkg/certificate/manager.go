package certificate

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/openservicemesh/osm/pkg/announcements"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/errcode"
	"github.com/openservicemesh/osm/pkg/k8s/events"
	"github.com/openservicemesh/osm/pkg/messaging"
)

var allowedMRCStateChanges = map[string][]string{
	constants.MRCStateValidatingRollout:  {constants.MRCStateIssuingRollout, constants.MRCStateError},
	constants.MRCStateIssuingRollout:     {constants.MRCStateActive, constants.MRCStateError},
	constants.MRCStateActive:             {constants.MRCStateIssuingRollback, constants.MRCStateError},
	constants.MRCStateIssuingRollback:    {constants.MRCStateValidatingRollback, constants.MRCStateError},
	constants.MRCStateValidatingRollback: {constants.MRCStateInactive, constants.MRCStateError},
}

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
		mrc := event.NewMRC
		if mrc.Status.State == constants.MRCStateError {
			log.Debug().Msgf("skipping MRC with error state %s", mrc.GetName())
			return nil
		}

		client, ca, clientID, err := mrcClient.GetCertIssuerForMRC(mrc)
		if err != nil {
			return err
		}

		c := &issuer{Issuer: client, ID: clientID, CertificateAuthority: ca}
		switch mrc.Status.State {
		case constants.MRCStateActive, constants.MRCStateValidatingRollback:
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
		default:
			m.mu.Lock()
			m.signingIssuer = c
			m.validatingIssuer = c
			m.mu.Unlock()
		}
	case MRCEventUpdated:
		// check if the update was a status change, ignore if not
		if event.OldMRC == nil || event.NewMRC == nil || event.OldMRC.Status.State == event.NewMRC.Status.State {
			log.Debug().Msgf("ignoring update event for MRC %s since status did not change", event.NewMRC.Name)
			return nil
		}

		return m.handleMRCStatusUpdate(mrcClient, event)
	}
	return nil
}

func (m *Manager) handleMRCStatusUpdate(mrcClient MRCClient, event MRCEvent) error {
	oldMRC := event.OldMRC
	newMRC := event.NewMRC

	if !ValidStateChange(oldMRC.Status.State, newMRC.Status.State) {
		msg := fmt.Sprintf("invalid state change for MRC %s: %s -> %s",
			newMRC.Name, oldMRC.Status.State, newMRC.Status.State)
		log.Error().Msg(msg)
		return errors.New(msg)
	}

	switch newMRC.Status.State {
	case constants.MRCStateValidatingRollout:
		// Should be handled by MRC added event

	case constants.MRCStateIssuingRollout:
		m.mu.Lock()
		validatingIssuerID := m.validatingIssuer.ID // Does this need to be locked?
		m.mu.Unlock()

		if newMRC.Name != validatingIssuerID {
			msg := fmt.Sprintf("expected MRC %s with updated status %s to be the current validatingIssuer %s",
				newMRC.Name, constants.MRCStateIssuingRollout, validatingIssuerID)
			log.Error().Msg(msg)
			return errors.New(msg)
		}

		m.mu.Lock()
		rollbackMRCID := m.signingIssuer.ID

		tempIssuer := m.signingIssuer
		m.signingIssuer = m.validatingIssuer
		m.validatingIssuer = tempIssuer
		m.mu.Unlock()

		_, err := mrcClient.Update(rollbackMRCID, newMRC.Namespace, constants.MRCStateIssuingRollback, nil)
		if apierrors.IsConflict(err) {
			log.Debug().Msgf("attempted to update MRC %s status to %s, but was already updated: %s",
				rollbackMRCID, constants.MRCStateIssuingRollback, err)
			return nil
		} else if err != nil {
			log.Error().Msgf("failed to update MRC %s status to %s: %s",
				rollbackMRCID, constants.MRCStateIssuingRollback, err)
			return err
		}

		log.Debug().Msgf("successfully updated status of MRC %s to %s", rollbackMRCID, constants.MRCStateIssuingRollback)
	case constants.MRCStateActive:
		m.mu.Lock()
		signingIssuerID := m.signingIssuer.ID
		m.mu.Unlock()

		if newMRC.Name != signingIssuerID {
			msg := fmt.Sprintf("expected MRC %s with updated status %s to be the current signingIssuer %s",
				newMRC.Name, constants.MRCStateActive, signingIssuerID)
			log.Error().Msg(msg)
			return errors.New(msg)
		}

		m.mu.Lock()
		rollbackMRCID := m.validatingIssuer.ID
		m.validatingIssuer = m.signingIssuer
		validatingIssuerID := m.validatingIssuer.ID
		m.mu.Unlock()

		if validatingIssuerID != signingIssuerID {
			msg := fmt.Sprintf("for MRC %s expected validatingIssuer ID %s to be the same as the signingIssuer ID %s",
				newMRC.Name, validatingIssuerID, signingIssuerID)
			log.Error().Msg(msg)
			return errors.New(msg)
		}

		_, err := mrcClient.Update(rollbackMRCID, newMRC.Namespace, constants.MRCStateValidatingRollback, nil)
		if apierrors.IsConflict(err) {
			log.Debug().Msgf("attempted to update MRC %s status to %s, but was already updated: %s",
				rollbackMRCID, constants.MRCStateIssuingRollback, err)
			break
		} else if err != nil {
			msg := fmt.Sprintf("failed to update MRC %s status to %s: %s",
				rollbackMRCID, constants.MRCStateIssuingRollback, err)
			log.Error().Msg(msg)
			return errors.New(msg)
		}

		log.Debug().Msgf("successfully updated status of MRC %s to %s", rollbackMRCID, constants.MRCStateIssuingRollback)
	case constants.MRCStateIssuingRollback, constants.MRCStateValidatingRollback:
		log.Info().Msgf("no operation required on %s status update to MRC %s", newMRC.Status.State, newMRC.Name)
	case constants.MRCStateInactive:
		// TODO(jaellio): delete MRC
	}
	return nil
}

func ValidStateChange(oldMRCStatus, newMRCStatus string) bool {
	allowedStates, ok := allowedMRCStateChanges[oldMRCStatus]
	if !ok {
		return false
	}
	for _, state := range allowedStates {
		if state == newMRCStatus {
			return true
		}
	}
	return false
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
