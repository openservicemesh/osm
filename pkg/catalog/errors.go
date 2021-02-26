package catalog

import "github.com/pkg/errors"

var (
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

	// ErrServiceNotFoundForAnyProvider is an error for when OSM cannot find a service for the given service account.
	ErrServiceNotFoundForAnyProvider = errors.New("no service found for service account with any of the mesh supported providers")

	// ErrNoTrafficSpecFoundForTrafficPolicy is an error for when OSM cannot find a traffic spec for the given traffic policy.
	ErrNoTrafficSpecFoundForTrafficPolicy = errors.New("no traffic spec found for the traffic policy")

	// ErrServiceNotFound is an error for when OSM cannot find a service.
	ErrServiceNotFound = errors.New("service not found")
)
