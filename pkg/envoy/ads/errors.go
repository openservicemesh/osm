package ads

import "fmt"

var errUnknownTypeURL = fmt.Errorf("unknown TypeUrl")
var errCreatingResponse = fmt.Errorf("creating response")
var errGrpcClosed = fmt.Errorf("grpc closed")
var errTooManyConnections = fmt.Errorf("too many connections")
var errServiceAccountMismatch = fmt.Errorf("service account mismatch in nodeid vs xds certificate common name")
var errUnsuportedXDSRequest = fmt.Errorf("Unsupported XDS server connection type")
var errInvalidCertificateCN = fmt.Errorf("invalid cn")
