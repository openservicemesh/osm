package identity

import "errors"

var (
	// ErrInvalidNamespacedServiceStringFormat is an error returned when the K8sServiceAccount string cannot be parsed (is invalid for some reason)
	ErrInvalidNamespacedServiceStringFormat = errors.New("invalid namespaced service string format")

	// ErrNotAKubernetesServiceAccount is an error returned when the ServiceIdentity is expected to be derived from a Kubernetes Service Account, but this is not true.
	ErrNotAKubernetesServiceAccount = errors.New("not a kubernetes service account based service identity")
)
