package kubernetes

import "github.com/pkg/errors"

var (
	errSyncingCaches     = errors.New("Failed initial cache sync for Namespace informers")
	errInitInformers     = errors.New("Informer not initialized")
	errListingNamespaces = errors.New("Failed to list monitored namespaces")
	errServiceNotFound   = errors.New("Service not found")

	// ErrInvalidCertificateCN is an error for when a certificate has a CommonName, which does not match expected string format.
	ErrInvalidCertificateCN = errors.New("invalid cn")

	// ErrMoreThanOnePodForCertificate is an error for when OSM finds more than one pod for a given xDS certificate. There should always be exactly one Pod for a given xDS certificate.
	ErrMoreThanOnePodForCertificate = errors.New("found more than one pod for xDS certificate")

	// ErrDidNotFindPodForCertificate is an error for when OSM cannot not find a pod for the given xDS certificate.
	ErrDidNotFindPodForCertificate = errors.New("did not find pod for certificate")

	// ErrServiceAccountDoesNotMatchCertificate is an error for when the service account of a Pod does not match the xDS certificate.
	ErrServiceAccountDoesNotMatchCertificate = errors.New("service account does not match certificate")

	// ErrNamespaceDoesNotMatchCertificate is an error for when the namespace of the Pod does not match the xDS certificate.
	ErrNamespaceDoesNotMatchCertificate = errors.New("namespace does not match certificate")
)
