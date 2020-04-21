package tresor

import (
	"errors"
)

var errEncodeKey = errors.New("encode key")
var errEncodeCert = errors.New("encode cert")
var errCreateCert = errors.New("create cert")
var errMarshalPrivateKey = errors.New("marshal private key")
var errGeneratingSerialNumber = errors.New("generate serial number")
var errGeneratingPrivateKey = errors.New("generate private")
var errNoIssuingCA = errors.New("no issuing CA")
var errDecodingPEMBlock = errors.New("failed to decode PEM block containing certificate")
var errInvalidFileName = errors.New("invalid filename")
var errNoCertificateInPEM = errors.New("no certificate in PEM")
var errNoPrivateKeyInPEM = errors.New("no private Key in PEM")
