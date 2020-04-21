package tresor

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"time"

	"github.com/pkg/errors"

	"github.com/open-service-mesh/osm/pkg/certificate"
)

// IssueCertificate implements certificate.Manager and returns a newly issued certificate.
func (cm *CertManager) IssueCertificate(cn certificate.CommonName) (certificate.Certificater, error) {
	log.Info().Msgf("Issuing new certificate for CN=%s", cn)
	if cert, exists := cm.cache[cn]; exists {
		log.Info().Msgf("Found in cache certificate with CN=%s", cn)
		return cert, nil
	}

	if cm.ca == nil || cm.ca.x509Cert == nil || cm.ca.rsaKey == nil {
		log.Error().Msgf("Invalid CA provided for issuance of certificate with CN=%s", cn)
		return nil, errNoCA
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
		DNSNames:     []string{string(cn)},
		Subject: pkix.Name{
			CommonName:   string(cn),
			Organization: []string{org},
		},
		NotBefore: now,
		NotAfter:  now.Add(cm.validity),

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, cm.ca.x509Cert, &certPrivKey.PublicKey, cm.ca.rsaKey)
	if err != nil {
		log.Error().Err(err).Msgf("Error issuing x509.CreateCertificate command for CN=%s", template.Subject.CommonName)
		return nil, errors.Wrap(err, errCreateCert.Error())
	}

	certPEM, err := encodeCertDERtoPEM(derBytes)
	if err != nil {
		log.Error().Err(err).Msgf("Error encoding certificate with CN=%s", template.Subject.CommonName)
		return nil, err
	}

	privKeyPEM, err := encodeKeyDERtoPEM(certPrivKey)
	if err != nil {
		log.Error().Err(err).Msgf("Error encoding private key for certificate with CN=%s", template.Subject.CommonName)
		return nil, err
	}

	cert := Certificate{
		commonName: string(cn),
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
