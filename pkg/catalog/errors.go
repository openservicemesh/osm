package catalog

import "github.com/pkg/errors"

var (
<<<<<<< HEAD
	errInvalidCertificateCN                  = errors.New("invalid cn")
	errMoreThanOnePodForCertificate          = errors.New("found more than one pod for certificate")
	errDidNotFindPodForCertificate           = errors.New("did not find pod for certificate")
	errServiceAccountDoesNotMatchCertificate = errors.New("service account does not match certificate")
	errNamespaceDoesNotMatchCertificate      = errors.New("namespace does not match certificate")
	errServiceNotFoundForAnyProvider         = errors.New("no service found for service account with any of the mesh supported providers")
	errNoServicesFoundForCertificate         = errors.New("no services found for certificate")
	errNoTrafficSpecFoundForTrafficPolicy    = errors.New("no traffic spec found for the traffic policy")
=======
	// ErrServiceNotFoundForAnyProvider is an error for when OSM cannot find a service for the given service account.
	ErrServiceNotFoundForAnyProvider = errors.New("no service found for service account with any of the mesh supported providers")

	// ErrNoTrafficSpecFoundForTrafficPolicy is an error for when OSM cannot find a traffic spec for the given traffic policy.
	ErrNoTrafficSpecFoundForTrafficPolicy = errors.New("no traffic spec found for the traffic policy")

	// ErrServiceNotFound is an error for when OSM cannot find a service.
	ErrServiceNotFound = errors.New("service not found")
>>>>>>> 865c66ed45ee888b5719d2e56a32f1534b61d1e7
)
