package certificate

import (
	"math/rand"
	"sync"
	time "time"

	"github.com/openservicemesh/osm/pkg/announcements"
	"github.com/openservicemesh/osm/pkg/errcode"
	"github.com/openservicemesh/osm/pkg/k8s/events"
	"github.com/openservicemesh/osm/pkg/messaging"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

const (
	// How much earlier (before expiration) should a certificate be renewed
	renewBeforeCertExpires = 30 * time.Second

	// So that we do not renew all certs at the same time - add noise.
	// These define the min and max of the seconds of noise to be added
	// to the early certificate renewal.
	minNoiseSeconds                    = 1
	maxNoiseSeconds                    = 5
	checkCertificateExpirationInterval = 5 * time.Second
)

var errCertNotFound = errors.New("failed to find cert")

type CertManager struct {
	provider                    Provider
	cache                       sync.Map
	msgBroker                   *messaging.Broker
	serviceCertValidityDuration time.Duration
}

func NewManager(provider Provider, serviceCertValidityDuration time.Duration) *CertManager {
	cm := &CertManager{
		provider: provider,
		//todo(schristoff) note about sync.Map
		msgBroker:                   messaging.NewBroker(make(<-chan struct{})),
		serviceCertValidityDuration: serviceCertValidityDuration,
	}
	cm.start(checkCertificateExpirationInterval)
	return cm
}

func (cm *CertManager) deleteFromCache(cn CommonName) {
	cm.cache.Delete(cn)
}

func (cm *CertManager) getFromCache(cn CommonName) *Certificate {
	if certInterface, exists := cm.cache.Load(cn); exists {
		cert := certInterface.(*Certificate)
		log.Trace().Msgf("Certificate found in cache SerialNumber=%s", cert.GetSerialNumber())
		if ShouldRotate(cert) {
			log.Trace().Msgf("Certificate found in cache but has expired SerialNumber=%s", cert.GetSerialNumber())
			return nil
		}
		return cert
	}
	return nil
}

// IssueCertificate implements Manager and returns a newly issued
func (cm *CertManager) IssueCertificate(cn CommonName, validityPeriod time.Duration) (*Certificate, error) {
	start := time.Now()

	if cert := cm.getFromCache(cn); cert != nil {
		return cert, nil
	}

	cert, err := cm.provider.IssueCertificate(cn, validityPeriod)
	if err != nil {
		return cert, err
	}

	cm.cache.Store(cn, cert)

	log.Trace().Msgf("It took %+v to issue certificate with SerialNumber=%s", time.Since(start), cert.GetSerialNumber())

	return cert, nil
}

// ReleaseCertificate is called when a cert will no longer be needed and should be removed from the system.
func (cm *CertManager) ReleaseCertificate(cn CommonName) {
	log.Trace().Msgf("Releasing certificate %s", cn)
	cm.deleteFromCache(cn)
}

// GetCertificate returns a certificate given its Common Name (CN)
func (cm *CertManager) GetCertificate(cn CommonName) (*Certificate, error) {
	if cert := cm.getFromCache(cn); cert != nil {
		return cert, nil
	}
	return nil, errCertNotFound
}

// RotateCertificate implements Manager and rotates an existing
func (cm *CertManager) RotateCertificate(cn CommonName) (*Certificate, error) {
	start := time.Now()

	oldObj, ok := cm.cache.Load(cn)
	if !ok {
		return nil, errors.Errorf("Old certificate does not exist for CN=%s", cn)
	}

	oldCert, ok := oldObj.(*Certificate)
	if !ok {
		return nil, errors.Errorf("unexpected type %T for old certificate does not exist for CN=%s", oldCert, cn)
	}

	newCert, err := cm.provider.IssueCertificate(cn, cm.serviceCertValidityDuration)
	if err != nil {
		return nil, err
	}

	cm.cache.Store(cn, newCert)

	cm.msgBroker.GetCertPubSub().Pub(events.PubSubMessage{
		Kind:   announcements.CertificateRotated,
		NewObj: newCert,
		OldObj: oldCert,
	}, announcements.CertificateRotated.String())

	log.Debug().Msgf("Rotated certificate (old SerialNumber=%s) with new SerialNumber=%s took %+v", oldCert.SerialNumber, newCert.SerialNumber, time.Since(start))

	return newCert, nil
}

// ListCertificates lists all certificates issued
func (cm *CertManager) ListCertificates() ([]*Certificate, error) {
	var certs []*Certificate
	cm.cache.Range(func(cn interface{}, certInterface interface{}) bool {
		certs = append(certs, certInterface.(*Certificate))
		return true // continue the iteration
	})
	return certs, nil
}

// GetRootCertificate returns the root
func (cm *CertManager) GetRootCertificate() (*Certificate, error) {
	return cm.provider.GetRootCertificate()
}

// Start starts a new facility for automatic certificate rotation.
func (cm *CertManager) start(checkInterval time.Duration) {
	// iterate over the list of certificates
	// when a cert needs to be rotated - call RotateCertificate()
	ticker := time.NewTicker(checkInterval)
	go func() {
		for {
			cm.checkAndRotate()
			<-ticker.C
		}
	}()
}

//todo(schristoff): point
func (cm *CertManager) checkAndRotate() {
	certs, err := cm.ListCertificates()
	if err != nil {
		log.Error().Err(err).Msgf("Error listing all certificates")
	}

	for _, cert := range certs {
		shouldRotate := ShouldRotate(cert)

		word := map[bool]string{true: "will", false: "will not"}[shouldRotate]
		log.Trace().Msgf("Cert %s %s be rotated; expires in %+v; renewBeforeCertExpires is %+v",
			cert.GetCommonName(),
			word,
			time.Until(cert.GetExpiration()),
			renewBeforeCertExpires)

		if shouldRotate {
			// Remove the certificate from the cache of the certificate manager
			newCert, err := cm.RotateCertificate(cert.GetCommonName())
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

// ShouldRotate determines whether a certificate should be rotated.
func ShouldRotate(cert *Certificate) bool {
	// The certificate is going to expire at a timestamp T
	// We want to renew earlier. How much earlier is defined in renewBeforeCertExpires.
	// We add a few seconds noise to the early renew period so that certificates that may have been
	// created at the same time are not renewed at the exact same time.

	intNoise := rand.Intn(maxNoiseSeconds-minNoiseSeconds) + minNoiseSeconds /* #nosec G404 */
	secondsNoise := time.Duration(intNoise) * time.Second
	return time.Until(cert.GetExpiration()) <= (renewBeforeCertExpires + secondsNoise)
}

// ListIssuedCertificates implements CertificateDebugger interface and returns the list of issued certificates.
func (cm *CertManager) ListIssuedCertificates() []*Certificate {
	var certs []*Certificate
	cm.cache.Range(func(cnInterface interface{}, certInterface interface{}) bool {
		certs = append(certs, certInterface.(*Certificate))
		return true // continue the iteration
	})
	return certs
}
