package service

import (
	"errors"
)

var (
	// ErrInvalidMeshServiceFormat indicates the format of the information used to populate
	// the MeshService is invalid
	ErrInvalidMeshServiceFormat = errors.New("invalid namespaced service string format")
)
