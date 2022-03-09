// Package bootstrap implements functionality related to Envoy's bootstrap config.
package bootstrap

import (
	"github.com/openservicemesh/osm/pkg/logger"
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

	// TrustedCA is the trusted certificate authority used to validate the certificate
	// presented by the XDS cluster during a TLS handshake
	TrustedCA []byte

	// CertificateChain is the certificate used by the proxy to connect to the XDS cluster
	CertificateChain []byte

	// PrivateKey is the private key for the certificate used by the proxy to connect to the XDS cluster
	PrivateKey []byte

	// TLSMinProtocolVersion is the minimum supported TLS protocol version
	TLSMinProtocolVersion string

	// TLSMaxProtocolVersion is the maximum supported TLS protocol version
	TLSMaxProtocolVersion string

	// CipherSuites is the list of cipher that TLS 1.0-1.2 supports
	CipherSuites []string

	// ECDHCurves is the list of ECDH curves it supports
	ECDHCurves []string
}
