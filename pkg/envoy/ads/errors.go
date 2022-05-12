package ads

import (
	"github.com/pkg/errors"
)

var errUnknownTypeURL = errors.New("unknown TypeUrl")
var errCreatingResponse = errors.New("creating response")
var errGrpcClosed = errors.New("grpc closed")
var errTooManyConnections = errors.New("too many connections")
var errServiceAccountMismatch = errors.New("service account mismatch in nodeid vs xds certificate common name")
var errUnsuportedXDSRequest = errors.New("Unsupported XDS server connection type")
