// Package bootstrap implements functionality related to Envoy's bootstrap config.
package bootstrap

import (
	"github.com/openservicemesh/osm/pkg/logger"
	"github.com/openservicemesh/osm/pkg/models"
)

var log = logger.New("envoy/bootstrap")

// Config is the type used to represent the information needed to build the Envoy bootstrap config
type Config struct {
	// Admin port is the Envoy admin port
	AdminPort uint32

	// XDSClusterName is the name of the XDS cluster to connect to
	XDSClusterName string

	// XDSHost is the hostname of the XDS cluster to connect to
	XDSHost string

	// XDSPort is the port of the XDS cluster to connect to
	XDSPort uint32

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
}

// EnvoyBootstrapConfigMeta contains context needed to compose the Envoy bootstrap YAML.
// TODO: remove this and leverage a simpler builder pattern.
type EnvoyBootstrapConfigMeta struct {
	EnvoyAdminPort uint32
	XDSClusterName string
	NodeID         string

	// Host and port of the Envoy xDS server
	XDSHost string
	XDSPort uint32

	// The bootstrap Envoy config will be affected by the liveness, readiness, startup probes set on
	// the pod this Envoy is fronting.
	OriginalHealthProbes models.HealthProbes

	// Sidecar TLS configuration
	TLSMinProtocolVersion string
	TLSMaxProtocolVersion string
	CipherSuites          []string
	ECDHCurves            []string
}
