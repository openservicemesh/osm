// Package providers implements generic certificate provider related functionality
package providers

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/openservicemesh/osm/pkg/configurator"
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

// Config is a type that stores config related to certificate providers and implements generic utility functions
type Config struct {
	kubeClient kubernetes.Interface
	kubeConfig *rest.Config
	cfg        configurator.Configurator

	providerKind       Kind
	providerNamespace  string
	caBundleSecretName string

	// tresorOptions is the options for 'Tresor' certificate provider
	tresorOptions TresorOptions

	// vaultOptions is the options for 'Hashicorp Vault' certificate provider
	vaultOptions VaultOptions

	// certManagerOptions is the options for 'cert-manager.io' certiticate provider
	certManagerOptions CertManagerOptions
}

// TresorOptions is a type that specifies 'Tresor' certificate provider options
type TresorOptions struct {
	// No options at the moment
}

// VaultOptions is a type that specifies 'Hashicorp Vault' certificate provider options
type VaultOptions struct {
	VaultProtocol string
	VaultHost     string
	VaultToken    string
	VaultRole     string
	VaultPort     int
}

// CertManagerOptions is a type that specifies 'cert-manager.io' certificate provider options
type CertManagerOptions struct {
	IssuerName  string
	IssuerKind  string
	IssuerGroup string
}
