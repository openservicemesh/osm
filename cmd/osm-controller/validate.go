package main

import (
	"strings"

	"github.com/pkg/errors"
)

// validateCLIParams contains all checks necessary that various permutations of the CLI flags are consistent
func validateCLIParams() error {
	if _, ok := certManagers[certificateManagerKind(*osmCertificateManagerKind)]; !ok {
		return errors.Errorf("Certificate manager %s is not one of possible options: %s", *osmCertificateManagerKind, strings.Join(getPossibleCertManagers(), ", "))
	}

	if *osmCertificateManagerKind == vaultKind {
		if *vaultToken == "" {
			return errors.Errorf("Empty Hashi Vault token")
		}
	}

	if *osmCertificateManagerKind == certmanagerKind {
		if len(caBundleSecretName) == 0 {
			return errors.Errorf("Please specify --%s as the Secret name containing the cert-manager CA at 'ca.crt'", caBundleSecretNameCLIParam)
		}
		if len(*certmanagerIssuerName) == 0 {
			return errors.Errorf("Please specify --cert-manager-issuer-name when using cert-manager certificate manager")
		}
	}

	if meshName == "" {
		return errors.Errorf("Please specify the mesh name using --mesh-name")
	}

	if osmNamespace == "" {
		return errors.Errorf("Please specify the OSM namespace using --osm-namespace")
	}

	if injectorConfig.InitContainerImage == "" {
		return errors.Errorf("Please specify the init container image using --init-container-image")
	}

	if injectorConfig.SidecarImage == "" {
		return errors.Errorf("Please specify the sidecar image using --sidecar-image")
	}

	if webhookName == "" {
		return errors.Errorf("Invalid --webhook-name value: '%s'", webhookName)
	}
	return nil
}
