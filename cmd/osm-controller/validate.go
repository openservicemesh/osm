package main

import (
	"github.com/pkg/errors"

	"github.com/openservicemesh/osm/pkg/certificate/providers"
)

// validateCLIParams contains all checks necessary that various permutations of the CLI flags are consistent
func validateCLIParams() error {
	if err := validateCertificateManagerOptions(); err != nil {
		return errors.Errorf("Error validating certificate manager options: %s", err)
	}

	if meshName == "" {
		return errors.New("Please specify the mesh name using --mesh-name")
	}

	if osmNamespace == "" {
		return errors.New("Please specify the OSM namespace using --osm-namespace")
	}

	if injectorConfig.InitContainerImage == "" {
		return errors.New("Please specify the init container image using --init-container-image")
	}

	if injectorConfig.SidecarImage == "" {
		return errors.Errorf("Please specify the sidecar image using --sidecar-image")
	}

	if webhookConfigName == "" {
		return errors.Errorf("Invalid --webhook-config-name value: '%s'", webhookConfigName)
	}

	if caBundleSecretName == "" {
		return errors.Errorf("Please specify the CA bundle secret name using --ca-bundle-secret-name containing the cert-manager CA at 'ca.crt'")
	}

	return nil
}

func validateCertificateManagerOptions() error {
	switch providers.Kind(certProviderKind) {
	case providers.TresorKind:
		return providers.ValidateTresorOptions(tresorOptions)

	case providers.VaultKind:
		return providers.ValidateVaultOptions(vaultOptions)

	case providers.CertManagerKind:
		return providers.ValidateCertManagerOptions(certManagerOptions)

	default:
		return errors.Errorf("Invalid certificate manager kind %s. Please specify a valid certificate manager, one of: [%v]",
			certProviderKind, providers.ValidCertificateProviders)
	}
}
