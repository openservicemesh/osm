package certificate

import (
	"bytes"
	"crypto/rsa"
	"crypto/x509"
	pemEnc "encoding/pem"

	"github.com/pkg/errors"

	"github.com/openservicemesh/osm/pkg/certificate/pem"
)

// EncodeCertDERtoPEM encodes the certificate provided in DER format into PEM format
// More information on the 2 formats is available in the following article: https://support.ssl.com/Knowledgebase/Article/View/19/0/der-vs-crt-vs-cer-vs-pem-certificates-and-how-to-convert-them
func EncodeCertDERtoPEM(derBytes []byte) (pem.Certificate, error) {
	certOut := &bytes.Buffer{}
	block := pemEnc.Block{
		Type:  TypeCertificate,
		Bytes: derBytes,
	}
	if err := pemEnc.Encode(certOut, &block); err != nil {
		return nil, errors.Wrap(err, errEncodeCert.Error())
	}
	return certOut.Bytes(), nil
}

// EncodeKeyDERtoPEM converts a DER encoded private key into a PEM encoded key
func EncodeKeyDERtoPEM(priv *rsa.PrivateKey) (pem.PrivateKey, error) {
	keyOut := &bytes.Buffer{}
	privBytes, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return nil, errors.Wrap(err, errMarshalPrivateKey.Error())
	}
	block := pemEnc.Block{
		Type:  TypePrivateKey,
		Bytes: privBytes,
	}
	if err := pemEnc.Encode(keyOut, &block); err != nil {
		return nil, errors.Wrap(err, errEncodeKey.Error())
	}
	return keyOut.Bytes(), nil
}

// DecodePEMCertificate converts a certificate from PEM to x509 encoding
func DecodePEMCertificate(certPEM []byte) (*x509.Certificate, error) {
	for len(certPEM) > 0 {
		var block *pemEnc.Block
		block, certPEM = pemEnc.Decode(certPEM)
		if block == nil {
			return nil, errNoCertificateInPEM
		}
		if block.Type != TypeCertificate || len(block.Headers) != 0 {
			continue
		}

		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			continue
		}

		return cert, nil
	}

	return nil, errNoCertificateInPEM
}

// DecodePEMPrivateKey converts a certificate from PEM to x509 encoding
func DecodePEMPrivateKey(keyPEM []byte) (*rsa.PrivateKey, error) {
	for len(keyPEM) > 0 {
		var block *pemEnc.Block
		block, keyPEM = pemEnc.Decode(keyPEM)
		if block == nil {
			return nil, errNoPrivateKeyInPEM
		}
		if block.Type != TypePrivateKey || len(block.Headers) != 0 {
			continue
		}

		caKeyInterface, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, err
		}
		return caKeyInterface.(*rsa.PrivateKey), nil
	}

	return nil, errNoCertificateInPEM
}

// EncodeCertReqDERtoPEM encodes the certificate request provided in DER format
// into PEM format.
func EncodeCertReqDERtoPEM(derBytes []byte) (pem.CertificateRequest, error) {
	csrPEM := bytes.NewBuffer([]byte{})
	block := pemEnc.Block{
		Type:  TypeCertificateRequest,
		Bytes: derBytes,
	}
	if err := pemEnc.Encode(csrPEM, &block); err != nil {
		return nil, errors.Wrap(err, errEncodeCert.Error())
	}
	return csrPEM.Bytes(), nil
}
