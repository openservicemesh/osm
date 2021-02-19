package certificate

import (
	"errors"
)

var (
	errEncodeKey          = errors.New("encode key")
	errEncodeCert         = errors.New("encode cert")
	errMarshalPrivateKey  = errors.New("marshal private key")
	errNoCertificateInPEM = errors.New("no certificate in PEM")
	errNoPrivateKeyInPEM  = errors.New("no private Key in PEM")
)
