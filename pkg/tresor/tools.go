package tresor

import (
	"bytes"
	"crypto/rsa"
	"crypto/x509"
	pemEnc "encoding/pem"

	"github.com/open-service-mesh/osm/pkg/tresor/pem"
	"github.com/pkg/errors"
)

func encodeCert(derBytes []byte) (pem.Certificate, error) {
	certOut := &bytes.Buffer{}
	if err := pemEnc.Encode(certOut, &pemEnc.Block{Type: TypeCertificate, Bytes: derBytes}); err != nil {
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
	if err := pemEnc.Encode(keyOut, &pemEnc.Block{Type: TypePrivateKey, Bytes: privBytes}); err != nil {
		return nil, errors.Wrap(err, errEncodeKey.Error())
	}
	return keyOut.Bytes(), nil
}
