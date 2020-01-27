package sds

import (
	"errors"
)

var errTooManyConnections = errors.New("too many connections")
var errEnvoyError = errors.New("Envoy error")
var errInvalidDiscoveryRequest = errors.New("invalid discovery request with no node")
var errGrpcClosed = errors.New("grpc closed")
