package tresor

import (
	"bytes"
	"crypto/rsa"
	"crypto/x509"
	pemEnc "encoding/pem"

	"github.com/pkg/errors"

	"github.com/open-service-mesh/osm/pkg/tresor/pem"
)

func encodeCert(derBytes []byte) (pem.Certificate, error) {
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

func encodeKey(priv *rsa.PrivateKey) (pem.PrivateKey, error) {
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
