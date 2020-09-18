package lds

import "github.com/pkg/errors"

var (
	errNoValidTargetEndpoints = errors.New("No valid resolvable addresses")
)
