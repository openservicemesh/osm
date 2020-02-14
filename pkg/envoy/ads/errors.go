package ads

import (
	"github.com/pkg/errors"
)

var errUnknownTypeURL = errors.New("unknown TypeUrl")
var errCreatingResponse = errors.New("creating response")
var errTooManyConnections = errors.New("too many connections")
var errEnvoyError = errors.New("Envoy error")
var errGrpcClosed = errors.New("grpc closed")
