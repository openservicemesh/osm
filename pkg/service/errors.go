package service

import (
	"errors"
)

var (
	errInvalidMeshServiceFormat = errors.New("invalid namespaced service string format")
)
