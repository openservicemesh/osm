package providers

import (
	"context"
	"fmt"
	"time"

	cmmeta "github.com/jetstack/cert-manager/pkg/apis/meta/v1"
	cmversionedclient "github.com/jetstack/cert-manager/pkg/client/clientset/versioned"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
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
	"github.com/openservicemesh/osm/pkg/version"
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

	certManager, certDebugger, err := config.GetCertificateManager()
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

// GetCertificateManager returns the certificate manager/provider instance
func (c *Config) GetCertificateManager() (certificate.Manager, debugger.CertificateManagerDebugger, error) {
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

// GetCertificateFromSecret is a helper function that ensures creation and synchronization of a certificate
// using Kubernetes Secrets backend and API atomicity.
func GetCertificateFromSecret(ns string, secretName string, cert certificate.Certificater, kubeClient kubernetes.Interface) (certificate.Certificater, error) {
	// Attempt to create it in Kubernetes. When multiple agents attempt to create, only one of them will succeed.
	// All others will get "AlreadyExists" error back.
	secretData := map[string][]byte{
		constants.KubernetesOpaqueSecretCAKey:             cert.GetCertificateChain(),
		constants.KubernetesOpaqueSecretCAExpiration:      []byte(cert.GetExpiration().Format(constants.TimeDateLayout)),
		constants.KubernetesOpaqueSecretRootPrivateKeyKey: cert.GetPrivateKey(),
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: ns,
			Labels: map[string]string{
				constants.OSMAppNameLabelKey:    constants.OSMAppNameLabelValue,
				constants.OSMAppVersionLabelKey: version.Version,
			},
		},
		Data: secretData,
	}

	if _, err := kubeClient.CoreV1().Secrets(ns).Create(context.TODO(), secret, metav1.CreateOptions{}); err == nil {
		log.Info().Msg("CA created in kubernetes")
	} else if apierrors.IsAlreadyExists(err) {
		log.Info().Msg("CA already exists in kubernetes, loading.")
	} else {
		log.Error().Err(err).Msgf("Error creating/retrieving root certificate from secret %s/%s", ns, secretName)
		return nil, err
	}

	// For simplicity, we will load the certificate for all of them, this way the intance which created it
	// and the ones that didn't share the same code.
	cert, err := GetCertFromKubernetes(ns, secretName, kubeClient)
	if err != nil {
		log.Error().Err(err).Msg("Failed to fetch certificate from Kubernetes")
		return nil, err
	}

	return cert, nil
}

// getTresorOSMCertificateManager returns a certificate manager instance with Tresor as the certificate provider
func (c *Config) getTresorOSMCertificateManager() (certificate.Manager, debugger.CertificateManagerDebugger, error) {
	var err error
	var rootCert certificate.Certificater

	// This part synchronizes CA creation using the inherent atomicity of kubernetes API backend
	// Assuming multiple instances of Tresor are instantiated at the same time, only one of them will
	// succeed to issue a "Create" of the secret. All other Creates will fail with "AlreadyExists".
	// Regardless of success or failure, all instances can proceed to load the same CA.

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

	rootCert, err = GetCertificateFromSecret(c.providerNamespace, c.caBundleSecretName, rootCert, c.kubeClient)
	if err != nil {
		return nil, nil, errors.Errorf("Failed to synchronize certificate on Secrets API : %v", err)
	}

	certManager, err := tresor.NewCertManager(rootCert, rootCertOrganization, c.cfg)
	if err != nil {
		return nil, nil, errors.Errorf("Failed to instantiate Tresor as a Certificate Manager")
	}

	return certManager, certManager, nil
}

// GetCertFromKubernetes is a helper function that loads a certificate from a Kubernetes secret
// The function returns an error only if a secret is found with invalid data.
func GetCertFromKubernetes(ns string, secretName string, kubeClient kubernetes.Interface) (certificate.Certificater, error) {
	certSecret, err := kubeClient.CoreV1().Secrets(ns).Get(context.Background(), secretName, metav1.GetOptions{})
	if err != nil {
		log.Debug().Msgf("Could not retrieve certificate secret %q from namespace %q", secretName, ns)
		return nil, errSecretNotFound
	}

	pemCert, ok := certSecret.Data[constants.KubernetesOpaqueSecretCAKey]
	if !ok {
		log.Error().Err(errInvalidCertSecret).Msgf("Opaque k8s secret %s/%s does not have required field %q", ns, secretName, constants.KubernetesOpaqueSecretCAKey)
		return nil, errInvalidCertSecret
	}

	pemKey, ok := certSecret.Data[constants.KubernetesOpaqueSecretRootPrivateKeyKey]
	if !ok {
		log.Error().Err(errInvalidCertSecret).Msgf("Opaque k8s secret %s/%s does not have required field %q", ns, secretName, constants.KubernetesOpaqueSecretRootPrivateKeyKey)
		return nil, errInvalidCertSecret
	}

	expirationBytes, ok := certSecret.Data[constants.KubernetesOpaqueSecretCAExpiration]
	if !ok {
		log.Error().Err(errInvalidCertSecret).Msgf("Opaque k8s secret %s/%s does not have required field %q", ns, secretName, constants.KubernetesOpaqueSecretCAExpiration)
		return nil, errInvalidCertSecret
	}

	expiration, err := time.Parse(constants.TimeDateLayout, string(expirationBytes))
	if err != nil {
		log.Error().Err(err).Msgf("Error parsing cert expiration %q from Kubernetes rootCertSecret %q from namespace %q", string(expirationBytes), secretName, ns)
		return nil, err
	}

	cert, err := tresor.NewCertificateFromPEM(pemCert, pemKey, expiration)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to create new Certificate from PEM")
		return nil, err
	}

	return cert, nil
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
