//go:build !fips

package constants

import "crypto/tls"

// MinTLSVersion is the minimum TLS version specified for control plane servers
const MinTLSVersion = tls.VersionTLS12
