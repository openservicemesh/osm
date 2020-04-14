package tresor

import (
	"bytes"
	"crypto/rsa"
	"crypto/x509"
	pemEnc "encoding/pem"

	"github.com/pkg/errors"

	"github.com/open-service-mesh/osm/pkg/tresor/pem"
)

// encodeCertDERtoPEM encodes the certificate provided in DER format into PEM format
// More information on the 2 formats is available in the following article: https://support.ssl.com/Knowledgebase/Article/View/19/0/der-vs-crt-vs-cer-vs-pem-certificates-and-how-to-convert-them
func encodeCertDERtoPEM(derBytes []byte) (pem.Certificate, error) {
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

// encodeKeyDERtoPEM converts a DER encoded private key into a PEM encoded key
func encodeKeyDERtoPEM(priv *rsa.PrivateKey) (pem.PrivateKey, error) {
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
