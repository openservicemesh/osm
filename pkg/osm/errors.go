package osm

import "fmt"

var errTooManyConnections = fmt.Errorf("too many connections")
var errInvalidCertificateCN = fmt.Errorf("invalid cn")
