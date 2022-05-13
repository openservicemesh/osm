package main

import (
	"github.com/pkg/errors"
)

// validateCLIParams contains all checks necessary that various permutations of the CLI flags are consistent
func validateCLIParams() error {
	if meshName == "" {
		return errors.New("Please specify the mesh name using --mesh-name")
	}

	if osmNamespace == "" {
		return errors.New("Please specify the OSM namespace using --osm-namespace")
	}

	if validatorWebhookConfigName == "" {
		return errors.Errorf("Please specify the webhook configuration name using --validator-webhook-config")
	}

	return nil
}
