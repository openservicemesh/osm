package certificate

import (
	"errors"
)

var errEncodeKey = errors.New("encode key")
var errEncodeCert = errors.New("encode cert")
var errMarshalPrivateKey = errors.New("marshal private key")
var errNoPrivateKeyInPEM = errors.New("no private Key in PEM")

// ErrNoCertificateInPEM is the errror for no certificate in PEM
var ErrNoCertificateInPEM = errors.New("no certificate in PEM")
