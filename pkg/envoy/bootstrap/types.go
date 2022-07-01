// Package bootstrap implements functionality related to Envoy's bootstrap config.
package bootstrap

import (
	"github.com/openservicemesh/osm/pkg/logger"
	"github.com/openservicemesh/osm/pkg/models"
)

var log = logger.New("envoy/bootstrap")

// Builder is the type used to build the Envoy bootstrap config.
type Builder struct {
	// XDSHost is the hostname of the XDS cluster to connect to
	XDSHost string

	// NodeID is the proxy's node ID
	NodeID string

	// TLSMinProtocolVersion is the minimum supported TLS protocol version
	TLSMinProtocolVersion string

	// TLSMaxProtocolVersion is the maximum supported TLS protocol version
	TLSMaxProtocolVersion string

	// CipherSuites is the list of cipher that TLS 1.0-1.2 supports
	CipherSuites []string

	// ECDHCurves is the list of ECDH curves it supports
	ECDHCurves []string

	OriginalHealthProbes models.HealthProbes
}
