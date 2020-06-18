package tests

import (
	"github.com/openservicemesh/osm/pkg/envoy"
	"strconv"

	xds_discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"google.golang.org/genproto/googleapis/rpc/status"
)

// NewDiscoveryRequestWithError creates a DiscoveryRequest with an error message.
func NewDiscoveryRequestWithError(typeURI envoy.TypeURI, respNonce, message string, versionInfo uint64) *xds_discovery.DiscoveryRequest {
	return &xds_discovery.DiscoveryRequest{
		TypeUrl:       typeURI.String(),
		VersionInfo:   strconv.FormatUint(versionInfo, 10),
		ResponseNonce: respNonce,
		ErrorDetail: &status.Status{
			Message: message,
		},
	}
}

// NewDiscoveryRequest creates a new DiscoveryRequest
func NewDiscoveryRequest(typeURI envoy.TypeURI, respNonce string, versionInfo uint64) *xds_discovery.DiscoveryRequest {
	return &xds_discovery.DiscoveryRequest{
		TypeUrl:       typeURI.String(),
		VersionInfo:   strconv.FormatUint(versionInfo, 10),
		ResponseNonce: respNonce,
	}
}
