package validator

import (
	"errors"
)

// ErrTrafficTargetNamespaceMismatch indicates that the namespace of the Traffic Target does not
//	match the destination namespace
var ErrTrafficTargetNamespaceMismatch = errors.New("traffic target namespace mismatch")
