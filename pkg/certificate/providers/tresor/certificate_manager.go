package tresor

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"time"

	"github.com/pkg/errors"

	"github.com/openservicemesh/osm/pkg/announcements"
	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/certificate/rotor"
)

func (cm *CertManager) issue(cn certificate.CommonName, validityPeriod time.Duration) (certificate.Certificater, error) {
	if cm.ca == nil {
		log.Error().Msgf("Invalid CA provided for issuance of certificate with CN=%s", cn)
		return nil, errNoIssuingCA
	}

	certPrivKey, err := rsa.GenerateKey(rand.Reader, rsaBits)
	if err != nil {
		log.Error().Err(err).Msgf("Error generating private key for certificate with CN=%s", cn)
		return nil, errors.Wrap(err, errGeneratingPrivateKey.Error())
	}

	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, errors.Wrap(err, errGeneratingSerialNumber.Error())
	}

	now := time.Now()
	template := x509.Certificate{
		SerialNumber: serialNumber,

		DNSNames: []string{string(cn)},

		Subject: pkix.Name{
			CommonName:   string(cn),
			Organization: []string{cm.certificatesOrganization},
		},
		NotBefore: now,
		NotAfter:  now.Add(validityPeriod),

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	x509Root, err := certificate.DecodePEMCertificate(cm.ca.GetCertificateChain())
	if err != nil {
		log.Error().Err(err).Msg("Error decoding Root Certificate's PEM")
	}

	rsaKeyRoot, err := certificate.DecodePEMPrivateKey(cm.ca.GetPrivateKey())
	if err != nil {
		log.Error().Err(err).Msg("Error decoding Root Certificate's Private Key PEM ")
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, x509Root, &certPrivKey.PublicKey, rsaKeyRoot)
	if err != nil {
		log.Error().Err(err).Msgf("Error issuing x509.CreateCertificate command for SerialNumber=%s", serialNumber)
		return nil, errors.Wrap(err, errCreateCert.Error())
	}

	certPEM, err := certificate.EncodeCertDERtoPEM(derBytes)
	if err != nil {
		log.Error().Err(err).Msgf("Error encoding certificate with SerialNumber=%s", serialNumber)
		return nil, err
	}

	privKeyPEM, err := certificate.EncodeKeyDERtoPEM(certPrivKey)
	if err != nil {
		log.Error().Err(err).Msgf("Error encoding private key for certificate with SerialNumber=%s", serialNumber)
		return nil, err
	}

	cert := Certificate{
		commonName:   cn,
		serialNumber: certificate.SerialNumber(serialNumber.String()),
		certChain:    certPEM,
		privateKey:   privKeyPEM,
		issuingCA:    cm.ca.GetCertificateChain(),
		expiration:   template.NotAfter,
	}

	log.Trace().Msgf("Created new certificate for SerialNumber=%s; validity=%+v; expires on %+v; serial: %x", serialNumber, validityPeriod, template.NotAfter, template.SerialNumber)

	return cert, nil
}

func (cm *CertManager) deleteFromCache(cn certificate.CommonName) {
	cm.cache.Delete(cn)
}

func (cm *CertManager) getFromCache(cn certificate.CommonName) certificate.Certificater {
	if certInterface, exists := cm.cache.Load(cn); exists {
		cert := certInterface.(certificate.Certificater)
		log.Trace().Msgf("Certificate found in cache SerialNumber=%s", cert.GetSerialNumber())
		if rotor.ShouldRotate(cert) {
			log.Trace().Msgf("Certificate found in cache but has expired SerialNumber=%s", cert.GetSerialNumber())
			return nil
		}
		return cert
	}
	return nil
}

// IssueCertificate implements certificate.Manager and returns a newly issued certificate.
func (cm *CertManager) IssueCertificate(cn certificate.CommonName, validityPeriod time.Duration) (certificate.Certificater, error) {
	start := time.Now()

	if cert := cm.getFromCache(cn); cert != nil {
		return cert, nil
	}

	cert, err := cm.issue(cn, validityPeriod)
	if err != nil {
		return cert, err
	}

	cm.cache.Store(cn, cert)

	log.Trace().Msgf("It took %+v to issue certificate with SerialNumber=%s", time.Since(start), cert.GetSerialNumber())

	return cert, nil
}

// ReleaseCertificate is called when a cert will no longer be needed and should be removed from the system.
func (cm *CertManager) ReleaseCertificate(cn certificate.CommonName) {
	log.Trace().Msgf("Releasing certificate %s", cn)
	cm.deleteFromCache(cn)
}

// GetCertificate returns a certificate given its Common Name (CN)
func (cm *CertManager) GetCertificate(cn certificate.CommonName) (certificate.Certificater, error) {
	if cert := cm.getFromCache(cn); cert != nil {
		return cert, nil
	}
	return nil, errCertNotFound
}

// RotateCertificate implements certificate.Manager and rotates an existing certificate.
func (cm *CertManager) RotateCertificate(cn certificate.CommonName) (certificate.Certificater, error) {
	start := time.Now()

	cert, err := cm.issue(cn, cm.cfg.GetServiceCertValidityPeriod())
	if err != nil {
		return cert, err
	}

	cm.cache.Store(cn, cert)
	cm.announcements <- announcements.Announcement{}

	log.Debug().Msgf("Rotated certificate with new SerialNumber=%s took %+v", cert.GetSerialNumber(), time.Since(start))

	return cert, nil
}

// ListCertificates lists all certificates issued
func (cm *CertManager) ListCertificates() ([]certificate.Certificater, error) {
	var certs []certificate.Certificater
	cm.cache.Range(func(cn interface{}, certInterface interface{}) bool {
		certs = append(certs, certInterface.(certificate.Certificater))
		return true // continue the iteration
	})
	return certs, nil
}

// GetRootCertificate returns the root certificate.
func (cm *CertManager) GetRootCertificate() (certificate.Certificater, error) {
	return cm.ca, nil
}

// GetAnnouncementsChannel implements certificate.Manager and returns the channel on which the certificate manager announces changes made to certificates.
func (cm *CertManager) GetAnnouncementsChannel() <-chan announcements.Announcement {
	return cm.announcements
}
