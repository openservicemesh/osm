package service

import (
	"errors"
)

var (
	errInvalidNamespacedServiceFormat = errors.New("invalid namespaced service string format")
)
