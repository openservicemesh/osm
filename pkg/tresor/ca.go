package tresor

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"time"
)

// NewCA creates a new Certificate Authority.
func NewCA(org string, validity time.Duration) (CA, CAPrivateKey, *x509.Certificate, *rsa.PrivateKey, error) {
	template, err := makeTemplate("", org, validity)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	template.IsCA = true
	template.KeyUsage |= x509.KeyUsageCertSign

	caPrivKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	caCert, caKey, err := genCert(template, template, caPrivKey, caPrivKey)
	return CA(caCert), CAPrivateKey(caKey), template, caPrivKey, err
}
