//go:build fips

package main

import _ "crypto/tls/fipsonly"

// This sole purpose of this file is to make sure FIPS configuration is enforced in this binary
