package sds

import "github.com/pkg/errors"

var errTooManyConnections = errors.New("too many connections")
var errEnvoyError = errors.New("Envoy error")
var errInvalidDiscoveryRequest = errors.New("invalid discovery request with no node")
var errKeyFileMissing = errors.New("key file is missing")
