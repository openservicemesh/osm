package certificate

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	time "time"

	"github.com/openservicemesh/osm/pkg/certificate/pem"
)

// CreateValidCertAndKey creates a non-expiring PEM certificate and private key
func CreateValidCertAndKey(cn CommonName, notBefore, notAfter time.Time) (pem.Certificate, pem.PrivateKey, error) {
	rootCertCountry := "US"
	rootCertLocality := "CA"
	rootCertOrganization := "Root Cert Organization"

	serialNumber := big.NewInt(1)

	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName:   cn.String(),
			Country:      []string{rootCertCountry},
			Locality:     []string{rootCertLocality},
			Organization: []string{rootCertOrganization},
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, template, template, &rsaKey.PublicKey, rsaKey)
	if err != nil {
		return nil, nil, err
	}
	pemCert, err := EncodeCertDERtoPEM(derBytes)
	if err != nil {
		return nil, nil, err
	}
	pemKey, err := EncodeKeyDERtoPEM(rsaKey)
	if err != nil {
		return nil, nil, err
	}
	return pemCert, pemKey, nil
}
