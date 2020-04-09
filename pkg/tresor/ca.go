package tresor

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"time"

	"github.com/pkg/errors"
)

// NewCA creates a new Certificate Authority.
func NewCA(validity time.Duration) (*Certificate, error) {
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, errors.Wrap(err, errGeneratingSerialNumber.Error())
	}

	now := time.Now()
	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName:   "Azure Mesh RSA Certification Authority",
			Country:      []string{"US"},
			Locality:     []string{"CA"},
			Organization: []string{org},
		},
		NotBefore:             now,
		NotAfter:              now.Add(validity),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	rsaKey, err := rsa.GenerateKey(rand.Reader, rsaBits)
	if err != nil {
		log.Error().Err(err).Msgf("Error generating key for CA for org %s", org)
		return nil, err
	}

	// Self-sign the root certificate
	derBytes, err := x509.CreateCertificate(rand.Reader, template, template, &rsaKey.PublicKey, rsaKey)
	if err != nil {
		log.Error().Err(err).Msgf("Error issuing x509.CreateCertificate command for CN=%s", template.Subject.CommonName)
		return nil, errors.Wrap(err, errCreateCert.Error())
	}

	pemCert, err := encodeCert(derBytes)
	if err != nil {
		log.Error().Err(err).Msgf("Error encoding certificate with CN=%s", template.Subject.CommonName)
		return nil, err
	}

	pemKey, err := encodeKey(rsaKey)
	if err != nil {
		log.Error().Err(err).Msgf("Error encoding private key for certificate with CN=%s", template.Subject.CommonName)
		return nil, err
	}

	rootCertificate := Certificate{
		name:       rootCertificateName,
		certChain:  pemCert,
		privateKey: pemKey,
		x509Cert:   template,
		rsaKey:     rsaKey,
	}

	return &rootCertificate, nil
}
