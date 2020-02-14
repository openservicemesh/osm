package envoy

import (
	"errors"
)

var ErrUnknownTypeURL = errors.New("unknown TypeUrl")
var ErrTooManyConnections = errors.New("too many connections")
