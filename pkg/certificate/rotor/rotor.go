package rotor

import (
	"math/rand"
	"time"

	"github.com/openservicemesh/osm/pkg/certificate"
)

const (
	// How much earlier (before expiration) should a certificate be renewed
	renewBeforeCertExpires = 30 * time.Second

	// So that we do not renew all certs at the same time - add noise.
	// These define the min and max of the seconds of noise to be added
	// to the early certificate renewal.
	minNoiseSeconds = 1
	maxNoiseSeconds = 5
)

type rotor struct {
	certificates *map[certificate.CommonName]certificate.Certificater
	certManager  certificate.Manager
}

// Start creates and starts a new facility for automatic certificate rotation.
func Start(checkInterval time.Duration, certManager certificate.Manager, certificates *map[certificate.CommonName]certificate.Certificater) {
	rtr := rotor{
		certificates: certificates,
		certManager:  certManager,
	}

	// TODO(draychev): current implementation is the naive one - we iterate over a list of certificates we are given
	// and if something needs to be rotated - we call the RotateCertificate function.
	ticker := time.NewTicker(checkInterval)
	go func() {
		for {
			rtr.checkAndRotate()
			<-ticker.C
		}
	}()
}

func (r *rotor) checkAndRotate() {
	for cn, cert := range *(r.certificates) {
		shouldRotate := ShouldRotate(cert)

		word := map[bool]string{true: "will", false: "will not"}[shouldRotate]
		log.Trace().Msgf("Cert %s %s be rotated; expires in %+v; renewBeforeCertExpires is %+v", cn, word, time.Until(cert.GetExpiration()), renewBeforeCertExpires)

		if shouldRotate {
			// Remove the certificate from the cache of the certificate manager
			newCert, err := r.certManager.RotateCertificate(cn)
			if err != nil {
				log.Error().Err(err).Msgf("Error rotating cert CN=%s", cn)
				continue
			}
			log.Trace().Msgf("Rotated cert CN=%s", newCert.GetCommonName())
		}
	}
}

// ShouldRotate determines whether a certificate should be rotated.
func ShouldRotate(cert certificate.Certificater) bool {
	// The certificate is going to expire at a timestamp T
	// We want to renew earlier. How much earlier is defined in renewBeforeCertExpires.
	// We add a few seconds noise to the early renew period so that certificates that may have been
	// created at the same time are not renewed at the exact same time.
	intNoise := rand.Intn(maxNoiseSeconds-minNoiseSeconds) + minNoiseSeconds
	secondsNoise := time.Duration(intNoise) * time.Second
	return time.Until(cert.GetExpiration()) <= (renewBeforeCertExpires + secondsNoise)
}
