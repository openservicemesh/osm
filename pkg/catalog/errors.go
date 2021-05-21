package catalog

import "github.com/pkg/errors"

var (
	// ErrServiceNotFoundForAnyProvider is an error for when OSM cannot find a service for the given service account.
	ErrServiceNotFoundForAnyProvider = errors.New("no service found for service account with any of the mesh supported providers")

	// ErrNoTrafficSpecFoundForTrafficPolicy is an error for when OSM cannot find a traffic spec for the given traffic policy.
	ErrNoTrafficSpecFoundForTrafficPolicy = errors.New("no traffic spec found for the traffic policy")

	// ErrServiceNotFound is an error for when OSM cannot find a service.
	ErrServiceNotFound = errors.New("service not found")
)
