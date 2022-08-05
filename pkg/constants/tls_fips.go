//go:build fips

package constants

import "crypto/tls"

const MinTLSVersion = tls.VersionTLS12
