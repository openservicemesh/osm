package main

import (
	"strings"

	"github.com/pkg/errors"
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

	return nil
}

func validateCertificateManagerOptions() error {
	switch *osmCertificateManagerKind {
	case tresorKind:
		break
	case vaultKind:
		if err := validateVaultParams(); err != nil {
			return err
		}
	case certmanagerKind:
		if err := validateCertManagerParams(); err != nil {
			return err
		}
	default:
		return errors.Errorf("Invalid certificate manager kind %s. Please specify a valid certificate manager[%v] \n",
			*osmCertificateManagerKind, strings.Join(validCertificateManagerOptions, "|"))
	}

	return nil
}

func validateCertManagerParams() error {
	if len(caBundleSecretName) == 0 {
		return errors.Errorf("Please specify --%s as the Secret name containing the cert-manager CA at 'ca.crt'", caBundleSecretNameCLIParam)
	}
	if len(*certmanagerIssuerName) == 0 {
		return errors.New("Please specify --cert-manager-issuer-name when using cert-manager certificate manager")
	}

	return nil
}

func validateVaultParams() error {
	if *vaultToken == "" {
		return errors.New("Empty Hashi Vault token")
	}

	return nil
}
