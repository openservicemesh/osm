package keyvault

import "errors"

var (
	errCertNotFound = errors.New("cert not found")

	errUnknownCertificateAuthorityType = errors.New("unknown certificate authority type")
)
