package ads

import "fmt"

var errUnknownTypeURL = fmt.Errorf("unknown TypeUrl")
var errTooManyConnections = fmt.Errorf("too many connections")
var errUnsuportedXDSRequest = fmt.Errorf("Unsupported XDS server connection type")
var errInvalidCertificateCN = fmt.Errorf("invalid cn")
