package tresor

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"time"

	"github.com/pkg/errors"
)

func encodeCert(derBytes []byte) (CertPEM, error) {
	certOut := &bytes.Buffer{}
	if err := pem.Encode(certOut, &pem.Block{Type: TypeCertificate, Bytes: derBytes}); err != nil {
		return nil, errors.Wrap(err, errEncodeCert.Error())
	}
	return certOut.Bytes(), nil
}

func encodeKey(priv *rsa.PrivateKey) (CertPrivKeyPEM, error) {
	keyOut := &bytes.Buffer{}
	privBytes, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return nil, errors.Wrap(err, errMarshalPrivateKey.Error())
	}
	if err := pem.Encode(keyOut, &pem.Block{Type: TypePrivateKey, Bytes: privBytes}); err != nil {
		return nil, errors.Wrap(err, errEncodeKey.Error())
	}
	return keyOut.Bytes(), nil
}

func makeTemplate(host string, org string, validity time.Duration) (*x509.Certificate, error) {
	// Validity duration of the certificate
	notBefore := time.Now()
	notAfter := notBefore.Add(validity)

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, errors.Wrap(err, errGeneratingSerialNumber.Error())
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		DNSNames:     []string{host},
		Subject: pkix.Name{
			CommonName:   host,
			Organization: []string{org},
		},
		NotBefore: notBefore,
		NotAfter:  notAfter,

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	return &template, nil
}
