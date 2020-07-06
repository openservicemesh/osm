package service

import (
	"errors"
)

var (
	errInvalidNamespacedServiceFormat = errors.New("invalid namespaced service string format")
	errInvalidCertFormat              = errors.New("invalid certificate string resource format")
)
