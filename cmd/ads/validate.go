package main

import (
	"fmt"
	"strings"
)

// validateCLIParams contains all checks necessary that various permutations of the CLI flags are consistent
func validateCLIParams() error {
	if _, ok := certManagers[certificateManagerKind(*certManagerKind)]; !ok {
		return fmt.Errorf("Certificate manager %s is not one of possible options: %s", *certManagerKind, strings.Join(getPossibleCertManagers(), ", "))
	}

	if *certManagerKind == vaultKind {
		if *vaultToken == "" {
			return fmt.Errorf("Empty Hashi Vault token")
		}
	}

	if meshName == "" {
		return fmt.Errorf("Please specify the mesh name using --mesh-name")
	}

	if osmNamespace == "" {
		return fmt.Errorf("Please specify the OSM namespace using --osm-namespace")
	}

	if injectorConfig.InitContainerImage == "" {
		return fmt.Errorf("Please specify the init container image using --init-container-image")
	}

	if injectorConfig.SidecarImage == "" {
		return fmt.Errorf("Please specify the sidecar image using --sidecar-image")
	}

	if webhookName == "" {
		return fmt.Errorf("Invalid --webhook-name value: '%s'", webhookName)
	}
	return nil
}
