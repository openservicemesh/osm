package tresor

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"time"

	"github.com/pkg/errors"

	"github.com/open-service-mesh/osm/pkg/tresor/pem"
)

// NewCA creates a new Certificate Authority.
func NewCA(org string, validity time.Duration) (pem.RootCertificate, pem.RootPrivateKey, *x509.Certificate, *rsa.PrivateKey, error) {
	// Validity duration of the certificate
	notBefore := time.Now()
	notAfter := notBefore.Add(validity)

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, nil, nil, nil, errors.Wrap(err, errGeneratingSerialNumber.Error())
	}

	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName:   "Azure Mesh RSA Certification Authority",
			Country:      []string{"US"},
			Locality:     []string{"CA"},
			Organization: []string{org},
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	caPrivKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		log.Error().Err(err).Msgf("Error generating key for CA for org %s", org)
		return nil, nil, nil, nil, err
	}

	caCert, caKey, err := genCert(template, template, caPrivKey, caPrivKey)
	if err != nil {
		log.Error().Err(err).Msgf("Error generating certificate for CA for org %s", org)
		return nil, nil, nil, nil, err
	}
	return pem.RootCertificate(caCert), pem.RootPrivateKey(caKey), template, caPrivKey, err
}
