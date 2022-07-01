// Package providers implements generic certificate provider related functionality
package providers

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"
	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/certificate/pem"
	"github.com/openservicemesh/osm/pkg/logger"
)

var log = logger.New("certificate/provider")

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

// Options is an interface that contains required fields to convert the old style options to the new style MRC for
// each provider type.
// TODO(#4502): Remove this interface, and all of the options below.
type Options interface {
	Validate() error

	AsProviderSpec() v1alpha2.ProviderSpec
}

// TresorOptions is a type that specifies 'Tresor' certificate provider options
type TresorOptions struct {
	// No options at the moment
	SecretName string
}

// VaultOptions is a type that specifies 'Hashicorp Vault' certificate provider options
type VaultOptions struct {
	VaultProtocol             string
	VaultHost                 string
	VaultToken                string // TODO(#4745): Remove after deprecating the osm.vault.token option. Replace with VaultTokenSecretName
	VaultRole                 string
	VaultPort                 int
	VaultTokenSecretNamespace string
	VaultTokenSecretName      string
	VaultTokenSecretKey       string
}

// CertManagerOptions is a type that specifies 'cert-manager.io' certificate provider options
type CertManagerOptions struct {
	IssuerName  string
	IssuerKind  string
	IssuerGroup string
}

// MRCCompatClient is a backwards compatible client to convert old certificate options into an MRC.
// It's intent is to match the custom interface that will wrap the MRC k8s informer.
// TODO(#4502): Remove this entirely once we are fully onboarded to MRC informers.
type MRCCompatClient struct {
	MRCProviderGenerator
	mrc *v1alpha2.MeshRootCertificate
}

// MRCProviderGenerator knows how to convert a given MRC to its appropriate provider.
type MRCProviderGenerator struct {
	kubeClient kubernetes.Interface
	kubeConfig *rest.Config // used to generate a CertificateManager client.

	// TODO(#4711): move these to the compat client once we have added these fields to the MRC.
	KeyBitSize int

	// TODO(#4745): Remove after deprecating the osm.vault.token option.
	DefaultVaultToken string
	caExtractorFunc   func(certificate.Issuer) (pem.RootCertificate, error)
}
