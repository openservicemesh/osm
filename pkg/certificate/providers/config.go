package providers

import (
	"context"
	"fmt"
	"time"

	cmmeta "github.com/jetstack/cert-manager/pkg/apis/meta/v1"
	cmversionedclient "github.com/jetstack/cert-manager/pkg/client/clientset/versioned"
	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/certificate/castorage/k8s"
	"github.com/openservicemesh/osm/pkg/certificate/providers/certmanager"
	"github.com/openservicemesh/osm/pkg/certificate/providers/tresor"
	"github.com/openservicemesh/osm/pkg/certificate/providers/vault"
	"github.com/openservicemesh/osm/pkg/certificate/rotor"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/debugger"
	"github.com/openservicemesh/osm/pkg/errcode"
	"github.com/openservicemesh/osm/pkg/messaging"
)

const (
	// Additional values for the root certificate
	rootCertCountry      = "US"
	rootCertLocality     = "CA"
	rootCertOrganization = "Open Service Mesh"

	checkCertificateExpirationInterval = 5 * time.Second
)

// GenerateCertificateManager returns a new certificate provider and associated config
func GenerateCertificateManager(kubeClient kubernetes.Interface, kubeConfig *rest.Config, cfg configurator.Configurator, providerKind Kind,
	providerNamespace string, caBundleSecretName string, tresorOptions TresorOptions, vaultOptions VaultOptions,
	certManagerOptions CertManagerOptions, msgBroker *messaging.Broker) (certificate.Manager, debugger.CertificateManagerDebugger, *Config, error) {
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

		msgBroker: msgBroker,

		caStorage: k8s.NewCASecretClient(kubeClient, caBundleSecretName, providerNamespace, ""),
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

// SetAndGetCA first attempts to write the CA, then reads it, even if the write succeeded.
func (c *Config) SetAndGetCA(ctx context.Context, cert *certificate.Certificate) (*certificate.Certificate, error) {
	// Attempt to create it in Kubernetes. When multiple agents attempt to create, only one of them will succeed.
	// All others will get "AlreadyExists" error back.
	if _, err := c.caStorage.Set(ctx, cert); errors.Is(err, certificate.ErrSecretAlreadyExists) {
		log.Info().Msgf("CA secret already exists in kubernetes, loading.")
	} else if err != nil {
		// TODO(#3962): metric might not be scraped before process restart resulting from this error
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrCreatingCertSecret)).
			Msgf("Error creating certificate secret")
		return nil, err
	} else {
		log.Info().Msgf("Secret created")
	}

	// For simplicity, we will load the certificate for all of them, this way the instance which created it
	// and the ones that didn't share the same code.
	cert, err := c.caStorage.Get(ctx)
	if err != nil {
		// TODO(#3962): metric might not be scraped before process restart resulting from this error
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrFetchingCertSecret)).
			Msgf("Could not retrieve certificate secret")
	}
	return cert, nil
}

// getTresorOSMCertificateManager returns a certificate manager instance with Tresor as the certificate provider
func (c *Config) getTresorOSMCertificateManager() (certificate.Manager, debugger.CertificateManagerDebugger, error) {
	var err error
	var rootCert *certificate.Certificate

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

	rootCert, err = c.SetAndGetCA(context.TODO(), rootCert)
	if err != nil {
		return nil, nil, err
	}

	if rootCert.GetPrivateKey() == nil {
		// TODO(#3962): metric might not be scraped before process restart resulting from this error
		log.Info().Err(certificate.ErrInvalidCertSecret).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrObtainingPrivateKeyFromSecret)).
			Msgf("Opaque k8s secret %s/%s does not have required field %q", c.providerNamespace, c.caBundleSecretName, constants.KubernetesOpaqueSecretRootPrivateKeyKey)
		return nil, nil, certificate.ErrInvalidCertSecret
	}

	tresorClient, err := tresor.New(
		rootCert,
		rootCertOrganization,
		c.cfg.GetCertKeyBitSize(),
	)
	if err != nil {
		return nil, nil, errors.Errorf("Failed to instantiate Tresor as a Certificate Manager")
	}

	tresorCertManager, err := certificate.NewManager(rootCert, tresorClient, c.cfg.GetServiceCertValidityPeriod(), c.msgBroker)
	if err != nil {
		return nil, nil, fmt.Errorf("error instantiating osm certificate.Manager for Tresor cert-manager : %w", err)
	}
	rotor.New(tresorCertManager).Start(checkCertificateExpirationInterval)

	return tresorCertManager, tresorCertManager, nil
}

// getHashiVaultOSMCertificateManager returns a certificate manager instance with Hashi Vault as the certificate provider
func (c *Config) getHashiVaultOSMCertificateManager(options VaultOptions) (certificate.Manager, debugger.CertificateManagerDebugger, error) {
	if _, ok := map[string]interface{}{"http": nil, "https": nil}[options.VaultProtocol]; !ok {
		return nil, nil, fmt.Errorf("value %s is not a valid Hashi Vault protocol", options.VaultProtocol)
	}

	// A Vault address would have the following shape: "http://vault.default.svc.cluster.local:8200"
	vaultAddr := fmt.Sprintf("%s://%s:%d", options.VaultProtocol, options.VaultHost, options.VaultPort)
	vaultClient, err := vault.New(
		vaultAddr,
		options.VaultToken,
		options.VaultRole,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("error instantiating Hashicorp Vault as a Certificate Manager: %w", err)
	}

	vaultCert, err := vaultClient.GetRootCertificate()
	if err != nil {
		return nil, nil, fmt.Errorf("error getting Vault Root Certificate, got: %w", err)
	}

	certManager, err := certificate.NewManager(vaultCert, vaultClient, c.cfg.GetServiceCertValidityPeriod(), c.msgBroker)
	if err != nil {
		return nil, nil, fmt.Errorf("error instantiating osm certificate.Manager for Vault cert-manager : %w", err)
	}

	rotor.New(certManager).Start(checkCertificateExpirationInterval)

	return certManager, certManager, nil
}

// getCertManagerOSMCertificateManager returns a certificate manager instance with cert-manager as the certificate provider
func (c *Config) getCertManagerOSMCertificateManager(options CertManagerOptions) (certificate.Manager, debugger.CertificateManagerDebugger, error) {
	rootCert, err := c.caStorage.Get(context.TODO())
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to get secret for certmanager.io: %w", err)
	}

	client, err := cmversionedclient.NewForConfig(c.kubeConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to build cert-manager client set: %s", err)
	}

	cmClient, err := certmanager.New(
		rootCert,
		client,
		c.providerNamespace,
		cmmeta.ObjectReference{
			Name:  options.IssuerName,
			Kind:  options.IssuerKind,
			Group: options.IssuerGroup,
		},
		c.cfg.GetCertKeyBitSize(),
	)
	if err != nil {
		return nil, nil, errors.Errorf("Error instantiating Jetstack cert-manager client: %+v", err)
	}

	certManager, err := certificate.NewManager(rootCert, cmClient, c.cfg.GetServiceCertValidityPeriod(), c.msgBroker)
	if err != nil {
		return nil, nil, errors.Errorf("error instantiating osm certificate.Manager for Jetstack cert-manager : %+v", err)
	}

	// TODO(#4533): push this into the certificate.manager object.
	rotor.New(certManager).Start(checkCertificateExpirationInterval)

	return certManager, certManager, nil
}
