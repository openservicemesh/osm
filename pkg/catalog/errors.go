package catalog

import "github.com/pkg/errors"

var (
	// errServiceNotFoundForAnyProvider is an error for when OSM cannot find a service for the given service account.
	errServiceNotFoundForAnyProvider = errors.New("no service found for service account with any of the mesh supported providers")

	// errNoTrafficSpecFoundForTrafficPolicy is an error for when OSM cannot find a traffic spec for the given traffic policy.
	errNoTrafficSpecFoundForTrafficPolicy = errors.New("no traffic spec found for the traffic policy")

	// errServiceNotFound is an error for when OSM cannot find a service.
	errServiceNotFound = errors.New("service not found")
)
