package rotor

import (
	"time"

	"github.com/open-service-mesh/osm/pkg/certificate"
)

const (
	tolerance         = 30 * time.Second
	maxSecondsShorter = 5
)

type rotor struct {
	certificates *map[certificate.CommonName]certificate.Certificater
	certManager  certificate.Manager

	// Announce to the outside world that a certificate has been rotated.
	announcements chan interface{}
}

// New creates and starts a new facility for automatic certificate rotation.
func New(checkInterval time.Duration, done chan interface{}, certManager certificate.Manager, certificates *map[certificate.CommonName]certificate.Certificater) <-chan interface{} {
	rtr := rotor{
		certificates:  certificates,
		certManager:   certManager,
		announcements: make(chan interface{}),
	}

	// TODO(draychev): current implementation is the naive one - we iterate over a list of certificates we are given
	// and if something needs to be rotated - we call the RotateCertificate function.
	ticker := time.NewTicker(checkInterval)
	go func() {
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				rtr.checkAndRotate()
			}
		}
	}()

	return rtr.announcements
}

func (r *rotor) checkAndRotate() {
	for cn, cert := range *(r.certificates) {
		shouldRotate := ShouldRotate(cert)

		word := map[bool]string{true: "will", false: "will not"}[shouldRotate]
		log.Trace().Msgf("Cert %s %s be rotated as it expires in %+v", cn, word, time.Until(cert.GetExpiration()))

		if shouldRotate {
			// Remove the certificate from the cache of the certificate manager
			newCert, err := r.certManager.RotateCertificate(cn)
			if err != nil {
				log.Error().Err(err).Msgf("Error rotating cert CN=%s", cn)
				continue
			}
			r.announcements <- nil
			log.Trace().Msgf("Rotated cert CN=%s", newCert.GetName())
		}
	}
}

// ShouldRotate determines whether a certificate should be rotated.
func ShouldRotate(cert certificate.Certificater) bool {
	expiresAt := cert.GetExpiration()
	randomSeconds := time.Duration(maxSecondsShorter) * time.Second
	return time.Until(expiresAt) < (tolerance - randomSeconds)
}
