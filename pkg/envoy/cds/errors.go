package cds

import (
	errors2 "errors"

	"github.com/pkg/errors"
)

var errTooManyConnections = errors.New("too many connections")
var errDiscoveryRequest = errors.New("discovery request error")
var errInvalidDiscoveryRequest = errors.New("invalid discovery request with no node")
var errGrpcClosed = errors2.New("grpc closed")
var errUnknownTypeURL = errors.New("unknown TypeUrl")
