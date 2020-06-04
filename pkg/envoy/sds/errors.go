package sds

import (
	"errors"
)

var (
	errInvalidResourceName      = errors.New("invalid resource name")
	errInvalidResourceKind      = errors.New("unknown resource kind")
	errInvalidResourceRequested = errors.New("invalid resource requested")
)
