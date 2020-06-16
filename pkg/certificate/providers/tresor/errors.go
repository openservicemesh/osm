package tresor

import (
	"errors"
)

var errCreateCert = errors.New("create cert")
var errGeneratingSerialNumber = errors.New("generate serial number")
var errGeneratingPrivateKey = errors.New("generate private")
var errNoIssuingCA = errors.New("no issuing CA")
var errCertNotFound = errors.New("certificate not found")
