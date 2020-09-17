package lds

import "github.com/pkg/errors"

var (
	errInvalidCIDRRange       = errors.New("invalid CIDR range")
	errNoValidTargetEndpoints = errors.New("No valid resolvable addresses")
)
