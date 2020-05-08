package sds

import (
	"errors"
)

var (
	errInvalidResourceName                     = errors.New("invalid resource name")
	errInvalidResourceKind                     = errors.New("unknown resource kind")
	errInvalidResourceRequested                = errors.New("invalid resource requested")
	errUnauthorizedRequestForServiceFromProxy  = errors.New("unauthorized request for service certificate from proxy")
	errUnauthorizedRequestForRootCertFromProxy = errors.New("unauthorized request for root certificate from proxy")
)
