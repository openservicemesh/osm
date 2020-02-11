package tresor

import (
	"errors"
)

var errInvalidHost = errors.New("invalid host")
var errEncodeKey = errors.New("encode key")
var errEncodeCert = errors.New("encode cert")
var errCreateCert = errors.New("create cert")
var errMarshalPrivateKey = errors.New("marshal private key")
var errGeneratingSerialNumber = errors.New("generate serial number")
var errGeneratingPrivateKey = errors.New("generate private")
var errNoCA = errors.New("no ca")
var errDecodingPEMBlock = errors.New("failed to decode PEM block containing certificate")
var errInvalidFileName = errors.New("invalid filename")
