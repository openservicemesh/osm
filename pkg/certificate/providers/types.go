// Package providers implements generic certificate provider related functionality
package providers

import (
	"github.com/openservicemesh/osm/pkg/logger"
)

var log = logger.New("cert-provider-util")

// Kind specifies the certificate provider kind
type Kind string

// String returns the Kind as a string
func (p Kind) String() string {
	return string(p)
}

const (
	// TresorKind represents Tresor, an internal package which leverages Kubernetes secrets and signs certs on the OSM pod
	TresorKind Kind = "tresor"

	// VaultKind represents Hashi Vault; OSM is pointed to an external Vault; signing of certs happens on Vault
	VaultKind Kind = "vault"

	// CertManagerKind represents cert-manager.io; certificates are requested using cert-manager
	CertManagerKind Kind = "cert-manager"
)

var (
	// ValidCertificateProviders is the list of supported certificate providers
	ValidCertificateProviders = []Kind{TresorKind, VaultKind, CertManagerKind}
)
