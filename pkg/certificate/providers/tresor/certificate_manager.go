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
	"github.com/openservicemesh/osm/pkg/errcode"
	"github.com/openservicemesh/osm/pkg/k8s/events"
)

func (cm *CertManager) issue(cn certificate.CommonName, validityPeriod time.Duration) (certificate.Certificater, error) {
	if cm.ca == nil {
		// TODO(#3962): metric might not be scraped before process restart resulting from this error
		log.Error().Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrInvalidCA)).
			Msgf("Invalid CA provided for issuance of certificate with CN=%s", cn)
		return nil, errNoIssuingCA
	}

	// Key bit size should remain static during the lifetime of the CertManager. In the event that this
	// is a zero value, we make the call to config to get the setting and then cache it for future
	// certificate operations.
	if cm.keySize == 0 {
		cm.keySize = cm.cfg.GetCertKeyBitSize()
	}
	certPrivKey, err := rsa.GenerateKey(rand.Reader, cm.keySize)
	if err != nil {
		// TODO(#3962): metric might not be scraped before process restart resulting from this error
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrGeneratingPrivateKey)).
			Msgf("Error generating private key for certificate with CN=%s", cn)
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
		// TODO(#3962): metric might not be scraped before process restart resulting from this error
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrDecodingPEMCert)).
			Msg("Error decoding Root Certificate's PEM")
	}

	rsaKeyRoot, err := certificate.DecodePEMPrivateKey(cm.ca.GetPrivateKey())
	if err != nil {
		// TODO(#3962): metric might not be scraped before process restart resulting from this error
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrDecodingPEMPrivateKey)).
			Msg("Error decoding Root Certificate's Private Key PEM ")
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, x509Root, &certPrivKey.PublicKey, rsaKeyRoot)
	if err != nil {
		// TODO(#3962): metric might not be scraped before process restart resulting from this error
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrCreatingCert)).
			Msgf("Error issuing x509.CreateCertificate command for SerialNumber=%s", serialNumber)
		return nil, errors.Wrap(err, errCreateCert.Error())
	}

	certPEM, err := certificate.EncodeCertDERtoPEM(derBytes)
	if err != nil {
		// TODO(#3962): metric might not be scraped before process restart resulting from this error
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrEncodingCertDERtoPEM)).
			Msgf("Error encoding certificate with SerialNumber=%s", serialNumber)
		return nil, err
	}

	privKeyPEM, err := certificate.EncodeKeyDERtoPEM(certPrivKey)
	if err != nil {
		// TODO(#3962): metric might not be scraped before process restart resulting from this error
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrEncodingKeyDERtoPEM)).
			Msgf("Error encoding private key for certificate with SerialNumber=%s", serialNumber)
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

	oldCert, ok := cm.cache.Load(cn)
	if !ok {
		return nil, errors.Errorf("Old certificate does not exist for CN=%s", cn)
	}

	// We want the validity duration of the CertManager to remain static during the lifetime
	// of the CertManager. This tests to see if this value is set, and if it isn't then it
	// should make the infrequent call to configuration to get this value and cache it for
	// future certificate operations.
	if cm.serviceCertValidityDuration == 0 {
		cm.serviceCertValidityDuration = cm.cfg.GetServiceCertValidityPeriod()
	}
	newCert, err := cm.issue(cn, cm.serviceCertValidityDuration)
	if err != nil {
		return nil, err
	}

	cm.cache.Store(cn, newCert)

	cm.msgBroker.GetCertPubSub().Pub(events.PubSubMessage{
		Kind:   announcements.CertificateRotated,
		NewObj: newCert,
		OldObj: oldCert.(certificate.Certificater),
	}, announcements.CertificateRotated.String())

	log.Debug().Msgf("Rotated certificate (old SerialNumber=%s) with new SerialNumber=%s took %+v", oldCert.(certificate.Certificater).GetSerialNumber(), newCert.GetSerialNumber(), time.Since(start))

	return newCert, nil
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
