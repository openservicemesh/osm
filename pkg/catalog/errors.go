package catalog

import "github.com/pkg/errors"

var (
	// errNoTrafficSpecFoundForTrafficPolicy is an error for when OSM cannot find a traffic spec for the given traffic policy.
	errNoTrafficSpecFoundForTrafficPolicy = errors.New("no traffic spec found for the traffic policy")
)
