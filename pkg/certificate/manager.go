package certificate

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/cskr/pubsub"

	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/errcode"
	"github.com/openservicemesh/osm/pkg/logger"
)

const (
	// MrcDurationPerStage is the amount of time we leave each MRC in a stage before moving to the next stage. This is
	// intended to accommodate the rotation of *all* rotations across all injector and controller pods for:
	// 1. Bootstrap Cert rotation for each live proxy
	// 2. Service Cert rotation and xDS push for each connected proxy
	// 3. xDS server cert rotation on the controller
	// 4. Mutating and Validating Webhook rotation for the servers and for the webhook configuration objects.
	// 5. Ingress Gateway Certificate.
	MrcDurationPerStage = 5 * time.Minute
)

var (
	log = logger.New("certificate")
)

// NewManager creates a new CertificateManager with the passed MRCClient and options
// TODO(5046): plumb ownedUseCases through.
func NewManager(ctx context.Context, mrcClient MRCClient, getServiceCertValidityPeriod func() time.Duration, getIngressCertValidityDuration func() time.Duration, checkInterval time.Duration) (*Manager, error) {
	m := &Manager{
		mrcClient:                   mrcClient,
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
				err = m.handleMRCEvent(event)
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

// GetTrustDomain returns the trust domain from the configured signingkey issuer.
// Note that the CRD uses a default, so this value will always be set.
func (m *Manager) GetTrustDomain() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.signingIssuer.TrustDomain
}

// ShouldRotate determines whether a certificate should be rotated.
func (m *Manager) ShouldRotate(c *Certificate) bool {
	// The certificate is going to expire at a timestamp T
	// We want to renew earlier. How much earlier is defined as the max between
	// a fractionValidityDuration of the certificate's validity duration and the minimum allowed time defined in minRotateBeforeExpireTime.
	// We add a few seconds noise to the early renew period so that certificates that may have been
	// created at the same time are not renewed at the exact same time.
	intNoise := rand.Intn(noiseSeconds) // #nosec G404
	secondsNoise := time.Duration(intNoise) * time.Second
	minRotateBeforeExpireTime := time.Duration(MinRotateBeforeExpireMinutes) * time.Minute
	fractionOfValidityDuration := m.getValidityDurationForCertType(c.certType) / fractionValidityDuration
	renewBefore := maxDuration(fractionOfValidityDuration, minRotateBeforeExpireTime) + secondsNoise
	// Round is called to truncate monotonic clock to the nearest second. This is done to avoid environments where the
	// CPU clock may stop, resulting in a time measurement that differs significantly from the x509 timestamp.
	// See https://github.com/openservicemesh/osm/issues/5000#issuecomment-1218539412 for more details.
	expiration := c.GetExpiration().Round(0)
	if time.Until(expiration) <= renewBefore {
		log.Info().Msgf("Cert %s should be rotated; expires in %+v; renewBefore is %+v",
			c.GetCommonName(),
			time.Until(expiration),
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
	log.Trace().Msgf("Cert %s should not be rotated with serial number %s and expiration %s", c.GetCommonName(), c.GetSerialNumber(), c.GetExpiration())
	return false
}

func maxDuration(a time.Duration, b time.Duration) time.Duration {
	if a >= b {
		return a
	}
	return b
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
		// TODO(5000): remove this check
		if err := m.CheckCacheMatch(cert); err != nil {
			log.Warn().Msg(err.Error()) // don't log as a full error message
		}
		opts := []IssueOption{}
		opts = append(opts, withCommonNamePrefix(key))
		opts = append(opts, withCertType(cert.certType))

		// There are a few certificates (webhook and Ingress)  that have entire CN passed
		// In that case the key will be the common name on the cert
		if key == cert.GetCommonName().String() {
			opts = append(opts, withFullCommonName())
		}

		_, err := m.IssueCertificate(opts...)
		if err != nil {
			log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrRotatingCert)).
				Msgf("Error rotating cert SerialNumber=%s", cert.GetSerialNumber())
		}
	}
}

func (m *Manager) getValidityDurationForCertType(ct certType) time.Duration {
	switch ct {
	case internal:
		return constants.OSMCertificateValidityPeriod
	case ingressGateway:
		return m.ingressCertValidityDuration()
	case service:
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
	log.Trace().Msgf("Certificate %s found in cache SerialNumber=%s", key, cert.GetSerialNumber())
	return cert
}

// IssueCertificate returns a newly issued certificate from the given client
// or an existing valid certificate from the local cache.
func (m *Manager) IssueCertificate(opts ...IssueOption) (*Certificate, error) {
	options := NewCertOptions(opts...)

	// a singleflight group is used here to ensure that only one issueCertificate is in
	// flight at a time for a given certificate prefix. Helps avoid a race condition if
	// issueCertificate is called multiple times in a row for the same certificate prefix.
	cert, err, _ := m.group.Do(options.cacheKey(), func() (interface{}, error) {
		return m.issueCertificate(options)
	})
	if err != nil {
		return nil, err
	}
	return cert.(*Certificate), nil
}

func (m *Manager) issueCertificate(options IssueOptions) (*Certificate, error) {
	var rotate bool
	cert := m.getFromCache(options.cacheKey()) // Don't call this while holding the lock
	if cert != nil {
		// check if cert needs to be rotated
		rotate = m.ShouldRotate(cert)
		if !rotate {
			return cert, nil
		}
	}

	m.mu.Lock()
	validatingIssuer := m.validatingIssuer
	signingIssuer := m.signingIssuer
	m.mu.Unlock()

	start := time.Now()

	options.ValidityDuration = m.getValidityDurationForCertType(options.certType)
	options.trustDomain = signingIssuer.TrustDomain
	options.spiffeEnabled = signingIssuer.SpiffeEnabled
	newCert, err := signingIssuer.IssueCertificate(options)
	if err != nil {
		return nil, err
	}

	// if we have different signing and validating issuers,
	// create the cert's trust context
	if validatingIssuer.ID != signingIssuer.ID {
		newCert = newCert.newMergedWithRoot(validatingIssuer.CertificateAuthority)
	}

	// Add some additional meta data for internal usage
	newCert.signingIssuerID = signingIssuer.ID
	newCert.validatingIssuerID = validatingIssuer.ID
	newCert.certType = options.certType
	newCert.cacheKey = options.cacheKey()

	m.cache.Store(newCert.cacheKey, newCert)

	log.Trace().Msgf("It took %s to issue certificate with SerialNumber=%s", time.Since(start), newCert.GetSerialNumber())

	if rotate {
		// Certificate was rotated
		m.pubsub.Pub(newCert, cert.cacheKey)

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

// CheckCacheMatch checks that the cert is the same cert present in the cache. it is currently only used for debugging
// https://github.com/openservicemesh/osm/issues/5000
// TODO(5000): delete this function once we fix the issue
func (m *Manager) CheckCacheMatch(cert *Certificate) error {
	if cert == nil {
		return nil
	}

	// Currently, these don't get rotated, so we assume the cache is correct.
	if cert.certType != service {
		return nil
	}
	key := strings.TrimSuffix(cert.CommonName.String(), "."+m.GetTrustDomain())
	cachedCert := m.getFromCache(key)
	if cachedCert == nil {
		return fmt.Errorf("no certificate found in cache for %s", cert.CommonName)
	}
	if cert != cachedCert {
		return fmt.Errorf("certificate %s does not match cached certificate %s, this may be due to a race around rotation, or a caching issue", cert, cachedCert)
	}
	return nil
}
