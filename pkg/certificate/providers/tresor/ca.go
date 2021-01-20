package tresor

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"time"

	"github.com/pkg/errors"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/certificate/pem"
)

// NewCA creates a new Certificate Authority.
func NewCA(cn certificate.CommonName, validityPeriod time.Duration, rootCertCountry, rootCertLocality, rootCertOrganization string) (certificate.Certificater, error) {
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, errors.Wrap(err, errGeneratingSerialNumber.Error())
	}

	now := time.Now()
	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName:   cn.String(),
			Country:      []string{rootCertCountry},
			Locality:     []string{rootCertLocality},
			Organization: []string{rootCertOrganization},
		},
		NotBefore:             now,
		NotAfter:              now.Add(validityPeriod),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	rsaKey, err := rsa.GenerateKey(rand.Reader, rsaBits)
	if err != nil {
		log.Error().Err(err).Msgf("Error generating key for CA for org %s", rootCertOrganization)
		return nil, err
	}

	// Self-sign the root certificate
	derBytes, err := x509.CreateCertificate(rand.Reader, template, template, &rsaKey.PublicKey, rsaKey)
	if err != nil {
		log.Error().Err(err).Msgf("Error issuing x509.CreateCertificate command for SerialNumber=%s", serialNumber)
		return nil, errors.Wrap(err, errCreateCert.Error())
	}

	pemCert, err := certificate.EncodeCertDERtoPEM(derBytes)
	if err != nil {
		log.Error().Err(err).Msgf("Error encoding certificate with SerialNumber=%s", serialNumber)
		return nil, err
	}

	pemKey, err := certificate.EncodeKeyDERtoPEM(rsaKey)
	if err != nil {
		log.Error().Err(err).Msgf("Error encoding private key for certificate with SerialNumber=%s", serialNumber)
		return nil, err
	}

	rootCertificate := Certificate{
		commonName:   rootCertificateName,
		serialNumber: certificate.SerialNumber(serialNumber.String()),
		certChain:    pemCert,
		privateKey:   pemKey,
		expiration:   template.NotAfter,
	}

	rootCertificate.issuingCA = rootCertificate.GetCertificateChain()

	return &rootCertificate, nil
}

// NewCertificateFromPEM is a helper returning a certificate.Certificater from the PEM components given.
func NewCertificateFromPEM(pemCert pem.Certificate, pemKey pem.PrivateKey, expiration time.Time) (certificate.Certificater, error) {
	x509Cert, err := certificate.DecodePEMCertificate(pemCert)
	if err != nil {
		log.Err(err).Msg("Error converting PEM cert to x509 to obtain serial number")
		return nil, err
	}
	rootCertificate := Certificate{
		commonName:   rootCertificateName,
		serialNumber: certificate.SerialNumber(x509Cert.SerialNumber.String()),
		certChain:    pemCert,
		privateKey:   pemKey,
		expiration:   expiration,
	}

	rootCertificate.issuingCA = rootCertificate.GetCertificateChain()

	return &rootCertificate, nil
}
