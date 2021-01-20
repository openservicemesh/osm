package certificate

import (
	"errors"
)

var errEncodeKey = errors.New("encode key")
var errEncodeCert = errors.New("encode cert")
var errMarshalPrivateKey = errors.New("marshal private key")
var errNoCertificateInPEM = errors.New("no certificate in PEM")
var errNoPrivateKeyInPEM = errors.New("no private Key in PEM")
