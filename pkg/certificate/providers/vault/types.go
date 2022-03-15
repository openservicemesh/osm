// Package vault implements the certificate.Manager interface for Hashicorp Vault as the certificate provider.
package vault

import (
	"github.com/hashicorp/vault/api"
)

// CertManager implements certificate.Manager and contains a Hashi Vault client instance.
type CertManager struct {
	// Hashicorp Vault client
	client *api.Client

	// The Vault role configured for OSM and passed as a CLI.
	role string
}
