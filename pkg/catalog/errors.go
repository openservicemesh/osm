package catalog

import "fmt"

var (
	// errNoTrafficSpecFoundForTrafficPolicy is an error for when OSM cannot find a traffic spec for the given traffic policy.
	errNoTrafficSpecFoundForTrafficPolicy = fmt.Errorf("no traffic spec found for the traffic policy")
)
