package identity

import "errors"

var (
	// ErrInvalidNamespacedServiceStringFormat is an error returned when the K8sServiceAccount string cannot be parsed (is invalid for some reason)
	ErrInvalidNamespacedServiceStringFormat = errors.New("invalid namespaced service string format")
)
