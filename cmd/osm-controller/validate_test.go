package main

import (
	"testing"

	tassert "github.com/stretchr/testify/assert"
)

func TestValidateCLIParams(t *testing.T) {
	testCases := []struct {
		name                       string
		meshName                   string
		osmNamespace               string
		validatorWebhookConfigName string
		expectError                bool
	}{
		{
			name:                       "none of the necessary CLI params are empty",
			meshName:                   "test-mesh",
			osmNamespace:               "test-ns",
			validatorWebhookConfigName: "test-webhook",
			expectError:                false,
		},
		{
			name:                       "mesh name is empty",
			meshName:                   "",
			osmNamespace:               "test-ns",
			validatorWebhookConfigName: "test-webhook",
			expectError:                true,
		},
		{
			name:                       "osm namespace is empty",
			meshName:                   "test-mesh",
			osmNamespace:               "",
			validatorWebhookConfigName: "test-webhook",
			expectError:                true,
		},
		{
			name:                       "validator webhook is empty",
			meshName:                   "test-mesh",
			osmNamespace:               "test-ns",
			validatorWebhookConfigName: "",
			expectError:                true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := tassert.New(t)
			meshName = tc.meshName
			osmNamespace = tc.osmNamespace
			validatorWebhookConfigName = tc.validatorWebhookConfigName
			err := validateCLIParams()
			assert.Equal(err != nil, tc.expectError)
		})
	}
}
