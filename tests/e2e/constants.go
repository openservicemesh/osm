package e2e

import "github.com/openservicemesh/osm/tests/framework"

const (
	fortioImageName = "fortio/fortio"
	fortioHTTPPort  = 8080
	fortioTCPPort   = 8078
	fortioGRPCPort  = 8079

	fortioTCPRetCodeSuccess  = "OK"
	fortioGRPCRetCodeSuccess = "SERVING"
)

var (
	fortioSingleCallSpec = framework.FortioLoadTestSpec{Calls: 1}
	// NumRetries is the number of retries for retry e2e
	NumRetries uint32 = 5
)
