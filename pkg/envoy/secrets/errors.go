package secrets

import (
	"errors"
)

var (
	errInvalidCertFormat                    = errors.New("invalid certificate string resource format")
	errInvalidMeshServiceFormat             = errors.New("invalid mesh service string format")
	errInvalidNamespacedServiceStringFormat = errors.New("invalid namespaced service string format")
)
