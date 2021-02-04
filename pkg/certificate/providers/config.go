package providers

import (
	"context"
	"fmt"
	"time"

	cmmeta "github.com/jetstack/cert-manager/pkg/apis/meta/v1"
	cmversionedclient "github.com/jetstack/cert-manager/pkg/client/clientset/versioned"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/certificate/providers/certmanager"
	"github.com/openservicemesh/osm/pkg/certificate/providers/tresor"
	"github.com/openservicemesh/osm/pkg/certificate/providers/vault"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/debugger"
)

const (
	// Additional values for the root certificate
	rootCertCountry      = "US"
	rootCertLocality     = "CA"
	rootCertOrganization = "Open Service Mesh"
)

// NewCertificateProvider returns a new certificate provider and associated config
func NewCertificateProvider(kubeClient kubernetes.Interface, kubeConfig *rest.Config, cfg configurator.Configurator, providerKind Kind,
	providerNamespace string, caBundleSecretName string, tresorOptions TresorOptions, vaultOptions VaultOptions,
	certManagerOptions CertManagerOptions) (certificate.Manager, debugger.CertificateManagerDebugger, *Config, error) {
	config := &Config{
		kubeClient:         kubeClient,
		kubeConfig:         kubeConfig,
		cfg:                cfg,
		providerKind:       providerKind,
		providerNamespace:  providerNamespace,
		caBundleSecretName: caBundleSecretName,

		tresorOptions:      tresorOptions,
		vaultOptions:       vaultOptions,
		certManagerOptions: certManagerOptions,
	}

	if err := config.Validate(); err != nil {
		return nil, nil, nil, err
	}

	certManager, certDebugger, err := config.getCertificateManager()
	if err != nil {
		return nil, nil, nil, err
	}

	return certManager, certDebugger, config, nil
}

// NewCertificateProviderConfig returns a new certificate provider config
func NewCertificateProviderConfig(kubeClient kubernetes.Interface, kubeConfig *rest.Config, cfg configurator.Configurator, providerKind Kind,
	providerNamespace string, caBundleSecretName string, tresorOptions TresorOptions, vaultOptions VaultOptions,
	certManagerOptions CertManagerOptions) *Config {
	return &Config{
		kubeClient:         kubeClient,
		kubeConfig:         kubeConfig,
		cfg:                cfg,
		providerKind:       providerKind,
		providerNamespace:  providerNamespace,
		caBundleSecretName: caBundleSecretName,

		tresorOptions:      tresorOptions,
		vaultOptions:       vaultOptions,
		certManagerOptions: certManagerOptions,
	}
}

// Validate validates the certificate provider config
func (c *Config) Validate() error {
	switch c.providerKind {
	case TresorKind:
		// nothing to validate
		return nil

	case VaultKind:
		return ValidateVaultOptions(c.vaultOptions)

	case CertManagerKind:
		return ValidateCertManagerOptions(c.certManagerOptions)

	default:
		return errors.Errorf("Invalid certificate manager kind %s. Specify a valid certificate manager, one of: [%v]",
			c.providerKind, ValidCertificateProviders)
	}
}

// ValidateTresorOptions validates the options for Tresor certificate provider
func ValidateTresorOptions(options TresorOptions) error {
	// Nothing to validate at the moment
	return nil
}

// ValidateVaultOptions validates the options for Hashi Vault certificate provider
func ValidateVaultOptions(options VaultOptions) error {
	if options.VaultHost == "" {
		return errors.New("VaultHost not specified in Hashi Vault options")
	}

	if options.VaultToken == "" {
		return errors.New("VaultToken not specified in Hashi Vault options")
	}

	if options.VaultRole == "" {
		return errors.New("VaultRole not specified in Hashi Vault options")
	}

	if _, ok := map[string]interface{}{"http": nil, "https": nil}[options.VaultProtocol]; !ok {
		return errors.Errorf("VaultProtocol in Hashi Vault options must be one of [http, https], got %s", options.VaultProtocol)
	}

	return nil
}

// ValidateCertManagerOptions validates the options for cert-manager.io certificate provider
func ValidateCertManagerOptions(options CertManagerOptions) error {
	if options.IssuerName == "" {
		return errors.New("IssuerName not specified in cert-manager.io options")
	}

	if options.IssuerKind == "" {
		return errors.New("IssuerKind not specified in cert-manager.io options")
	}

	if options.IssuerGroup == "" {
		return errors.New("IssuerGroup not specified in cert-manager.io options")
	}

	return nil
}

// getCertificateManager returns the certificate manager/provider instance
func (c *Config) getCertificateManager() (certificate.Manager, debugger.CertificateManagerDebugger, error) {
	switch c.providerKind {
	case TresorKind:
		return c.getTresorOSMCertificateManager()
	case VaultKind:
		return c.getHashiVaultOSMCertificateManager(c.vaultOptions)
	case CertManagerKind:
		return c.getCertManagerOSMCertificateManager(c.certManagerOptions)
	default:
		return nil, nil, fmt.Errorf("Unsupported Certificate Manager %s", c.providerKind)
	}
}

// getTresorOSMCertificateManager returns a certificate manager instance with Tresor as the certificate provider
func (c *Config) getTresorOSMCertificateManager() (certificate.Manager, debugger.CertificateManagerDebugger, error) {
	var err error
	var rootCert certificate.Certificater

	// The caBundleSecretName indicates to the certificate issuer to
	// load the CA from the given k8s secret within the namespace where OSM is install.d
	rootCert, err = c.GetCertFromKubernetes()
	if err != nil {
		log.Error().Err(err).Msgf("Error retrieving root certificate from secret %s/%s", c.providerNamespace, c.caBundleSecretName)
		return nil, nil, err
	}

	if rootCert == nil {
		rootCert, err = tresor.NewCA(constants.CertificationAuthorityCommonName, constants.CertificationAuthorityRootValidityPeriod, rootCertCountry, rootCertLocality, rootCertOrganization)

		if err != nil {
			return nil, nil, errors.Errorf("Failed to create new Certificate Authority with cert issuer %s", c.providerKind)
		}

		if rootCert == nil {
			return nil, nil, errors.Errorf("Invalid root certificate created by cert issuer %s", c.providerKind)
		}

		if rootCert.GetPrivateKey() == nil {
			return nil, nil, errors.Errorf("Root cert does not have a private key")
		}
	}

	certManager, err := tresor.NewCertManager(rootCert, rootCertOrganization, c.cfg)
	if err != nil {
		return nil, nil, errors.Errorf("Failed to instantiate Tresor as a Certificate Manager")
	}

	return certManager, certManager, nil
}

// GetCertFromKubernetes returns a Certificater type corresponding to the root certificate.
// The function returns an error only if a secret is found with invalid data.
func (c *Config) GetCertFromKubernetes() (certificate.Certificater, error) {
	rootCertSecret, err := c.kubeClient.CoreV1().Secrets(c.providerNamespace).Get(context.Background(), c.caBundleSecretName, metav1.GetOptions{})
	if err != nil {
		// It is okay for this secret to be missing, in which case a new CA will be created along with a k8s secret
		log.Debug().Msgf("Could not retrieve root certificate secret %q from namespace %q", c.caBundleSecretName, c.providerNamespace)
		return nil, nil
	}

	pemCert, ok := rootCertSecret.Data[constants.KubernetesOpaqueSecretCAKey]
	if !ok {
		log.Error().Err(errInvalidCertSecret).Msgf("Opaque k8s secret %s/%s does not have required field %q", c.providerNamespace, c.caBundleSecretName, constants.KubernetesOpaqueSecretCAKey)
		return nil, errInvalidCertSecret
	}

	pemKey, ok := rootCertSecret.Data[constants.KubernetesOpaqueSecretRootPrivateKeyKey]
	if !ok {
		log.Error().Err(errInvalidCertSecret).Msgf("Opaque k8s secret %s/%s does not have required field %q", c.providerNamespace, c.caBundleSecretName, constants.KubernetesOpaqueSecretRootPrivateKeyKey)
		return nil, errInvalidCertSecret
	}

	expirationBytes, ok := rootCertSecret.Data[constants.KubernetesOpaqueSecretCAExpiration]
	if !ok {
		log.Error().Err(errInvalidCertSecret).Msgf("Opaque k8s secret %s/%s does not have required field %q", c.providerNamespace, c.caBundleSecretName, constants.KubernetesOpaqueSecretCAExpiration)
		return nil, errInvalidCertSecret
	}

	expiration, err := time.Parse(constants.TimeDateLayout, string(expirationBytes))
	if err != nil {
		log.Error().Err(err).Msgf("Error parsing CA expiration %q from Kubernetes rootCertSecret %q from namespace %q", string(expirationBytes), c.caBundleSecretName, c.providerNamespace)
		return nil, err
	}

	rootCert, err := tresor.NewCertificateFromPEM(pemCert, pemKey, expiration)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to create new Certificate Authority with cert issuer %s", c.providerKind)
		return nil, err
	}

	return rootCert, nil
}

// getHashiVaultOSMCertificateManager returns a certificate manager instance with Hashi Vault as the certificate provider
func (c *Config) getHashiVaultOSMCertificateManager(options VaultOptions) (certificate.Manager, debugger.CertificateManagerDebugger, error) {
	if _, ok := map[string]interface{}{"http": nil, "https": nil}[options.VaultProtocol]; !ok {
		return nil, nil, errors.Errorf("Value %s is not a valid Hashi Vault protocol", options.VaultProtocol)
	}

	// A Vault address would have the following shape: "http://vault.default.svc.cluster.local:8200"
	vaultAddr := fmt.Sprintf("%s://%s:%d", options.VaultProtocol, options.VaultHost, options.VaultPort)
	vaultCertManager, err := vault.NewCertManager(vaultAddr, options.VaultToken, options.VaultRole, c.cfg)
	if err != nil {
		return nil, nil, errors.Errorf("Error instantiating Hashicorp Vault as a Certificate Manager: %+v", err)
	}

	return vaultCertManager, vaultCertManager, nil
}

// getCertManagerOSMCertificateManager returns a certificate manager instance with cert-manager as the certificate provider
func (c *Config) getCertManagerOSMCertificateManager(options CertManagerOptions) (certificate.Manager, debugger.CertificateManagerDebugger, error) {
	rootCertSecret, err := c.kubeClient.CoreV1().Secrets(c.providerNamespace).Get(context.TODO(), c.caBundleSecretName, metav1.GetOptions{})
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to get cert-manager CA secret %s/%s: %s", c.providerNamespace, c.caBundleSecretName, err)
	}

	pemCert, ok := rootCertSecret.Data[constants.KubernetesOpaqueSecretCAKey]
	if !ok {
		return nil, nil, fmt.Errorf("Opaque k8s secret %s/%s does not have required field %q", c.providerNamespace, c.caBundleSecretName, constants.KubernetesOpaqueSecretCAKey)
	}

	rootCert, err := certmanager.NewRootCertificateFromPEM(pemCert)
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to decode cert-manager CA certificate from secret %s/%s: %s", c.providerNamespace, c.caBundleSecretName, err)
	}

	client, err := cmversionedclient.NewForConfig(c.kubeConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to build cert-manager client set: %s", err)
	}

	certmanagerCertManager, err := certmanager.NewCertManager(rootCert, client, c.providerNamespace, cmmeta.ObjectReference{
		Name:  options.IssuerName,
		Kind:  options.IssuerKind,
		Group: options.IssuerGroup,
	}, c.cfg)
	if err != nil {
		return nil, nil, errors.Errorf("Error instantiating Jetstack cert-manager as a Certificate Manager: %+v", err)
	}

	return certmanagerCertManager, certmanagerCertManager, nil
}
