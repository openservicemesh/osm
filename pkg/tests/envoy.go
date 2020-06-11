package tests

import (
	"strconv"

	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"google.golang.org/genproto/googleapis/rpc/status"
)

// NewDiscoveryRequestWithError creates a DiscoveryRequest with an error message.
func NewDiscoveryRequestWithError(typeURI, respNonce, message string, versionInfo uint64) *v2.DiscoveryRequest {
	return &v2.DiscoveryRequest{
		TypeUrl:       typeURI,
		VersionInfo:   strconv.FormatUint(versionInfo, 10),
		ResponseNonce: respNonce,
		ErrorDetail: &status.Status{
			Message: message,
		},
	}
}

// NewDiscoveryRequest creates a new DiscoveryRequest
func NewDiscoveryRequest(typeURI, respNonce string, versionInfo uint64) *v2.DiscoveryRequest {
	return &v2.DiscoveryRequest{
		TypeUrl:       typeURI,
		VersionInfo:   strconv.FormatUint(versionInfo, 10),
		ResponseNonce: respNonce,
	}
}
