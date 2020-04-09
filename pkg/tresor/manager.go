package tresor

import (
	"crypto/rand"
	"crypto/rsa"

	"github.com/pkg/errors"

	"github.com/open-service-mesh/osm/pkg/certificate"
)

// IssueCertificate implements certificate.Manager and returns a newly issued certificate.
func (cm *CertManager) IssueCertificate(cn certificate.CommonName) (certificate.Certificater, error) {
	log.Info().Msgf("Issuing new certificate for CN=%s", cn)
	if cert, exists := cm.cache[cn]; exists {
		log.Info().Msgf("Found in cache - certificate with CN=%s", cn)
		return cert, nil
	}

	if cm.ca == nil || cm.ca.rsaKey == nil {
		log.Error().Msgf("Invalid CA provided for issuance of certificate with CN=%s", cn)
		return nil, errNoCA
	}

	certPrivKey, err := rsa.GenerateKey(rand.Reader, rsaBits)
	if err != nil {
		log.Error().Err(err).Msgf("Error generating private key for certificate with CN=%s", cn)
		return nil, errors.Wrap(err, errGeneratingPrivateKey.Error())
	}

	template, err := makeTemplate(string(cn), org, cm.validity)
	if err != nil {
		log.Error().Err(err).Msgf("Error creating template for certificate with CN=%s", cn)
		return nil, err
	}

	certPEM, privKeyPEM, err := genCert(template, cm.ca.x509Cert, certPrivKey, cm.ca.rsaKey)
	if err != nil {
		log.Error().Err(err).Msgf("Error creating certificate with CN=%s", cn)
		return nil, err
	}
	cert := Certificate{
		name:       string(cn),
		certChain:  certPEM,
		privateKey: privKeyPEM,
		ca:         cm.ca,
	}
	cm.cache[cn] = cert
	return cert, nil
}

// GetAnnouncementsChannel implements certificate.Manager and returns the channel on which the certificate manager announces changes made to certificates.
func (cm CertManager) GetAnnouncementsChannel() <-chan interface{} {
	return cm.announcements
}
