package rotor

import (
	"time"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/errcode"
)

// New creates and starts a new facility for automatic certificate rotation.
func New(certManager certificate.Manager) *CertRotor {
	return &CertRotor{
		certManager: certManager,
	}
}

// Start starts a new facility for automatic certificate rotation.
func (r CertRotor) Start(checkInterval time.Duration) {
	// iterate over the list of certificates
	// when a cert needs to be rotated - call RotateCertificate()
	ticker := time.NewTicker(checkInterval)
	go func() {
		for {
			r.checkAndRotate()
			<-ticker.C
		}
	}()
}

func (r *CertRotor) checkAndRotate() {
	certs, err := r.certManager.ListCertificates()
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
			certificate.RenewBeforeCertExpires)

		if shouldRotate {
			// Remove the certificate from the cache of the certificate manager
			newCert, err := r.certManager.RotateCertificate(cert.GetCommonName())
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
<<<<<<< HEAD

// ShouldRotate determines whether a certificate should be rotated.
func ShouldRotate(cert certificate.Certificater) bool {
	// The certificate is going to expire at a timestamp T
	// We want to renew earlier. How much earlier is defined in renewBeforeCertExpires.
	// We add a few seconds noise to the early renew period so that certificates that may have been
	// created at the same time are not renewed at the exact same time.

	intNoise := rand.Intn(maxNoiseSeconds-minNoiseSeconds) + minNoiseSeconds /* #nosec G404 */
	secondsNoise := time.Duration(intNoise) * time.Second
	return time.Until(cert.GetExpiration()) <= (renewBeforeCertExpires + secondsNoise)
}
=======
>>>>>>> caaa189c (feat(certificates) begin to abstract the cert manager patterns (#4580))
