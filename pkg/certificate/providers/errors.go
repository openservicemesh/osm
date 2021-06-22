package providers

import (
	"github.com/pkg/errors"
)

var (
	errInvalidCertSecret = errors.New("Invalid secret for certificate")
	errSecretNotFound    = errors.Errorf("Secret not found")
)
