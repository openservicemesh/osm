package tests

import (
	"crypto/tls"
	"crypto/x509"

	"google.golang.org/grpc/credentials"
)

// NewMockAuthInfo creates a new credentials.AuthInfo
func NewMockAuthInfo(cert *x509.Certificate) credentials.AuthInfo {
	return credentials.TLSInfo{
		State: tls.ConnectionState{
			ServerName:     "server.name",
			VerifiedChains: [][]*x509.Certificate{{cert}}},
	}
}
