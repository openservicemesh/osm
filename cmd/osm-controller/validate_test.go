package main

import (
	"testing"

	tassert "github.com/stretchr/testify/assert"

	"github.com/openservicemesh/osm/pkg/certificate/providers"
)

func TestValidateCertificateManagerOptions(t *testing.T) {
	testCases := []struct {
		name               string
		certProvider       string
		vaultToken         string
		caBundleSecretName string
		issuerName         string
		expectError        bool
	}{
		{
			name:         "Cert Provider : Tresor",
			certProvider: providers.TresorKind.String(),
			expectError:  false,
		},
		{
			name:         "Cert Provider : Vault and token is not empty",
			certProvider: providers.VaultKind.String(),
			vaultToken:   "anythinghere",
			expectError:  false,
		},
		{
			name:         "Cert Provider : Vault and token is empty",
			certProvider: providers.VaultKind.String(),
			vaultToken:   "",
			expectError:  true,
		},
		{
			name:               "Cert Provider : Cert-Manager with valid caBundleSecretName and certmanagerIssuerName",
			certProvider:       providers.CertManagerKind.String(),
			caBundleSecretName: "test-secret",
			issuerName:         "test-issuer",
			expectError:        false,
		},
		{
			name:               "Cert Provider : Cert-Manager with valid caBundleSecretName and no certmanagerIssuerName",
			certProvider:       providers.CertManagerKind.String(),
			caBundleSecretName: "test-secret",
			issuerName:         "",
			expectError:        true,
		},
		{
			name:               "Cert Provider : Cert-Manager with no caBundleSecretName and no certmanagerIssuerName",
			certProvider:       providers.CertManagerKind.String(),
			issuerName:         "",
			caBundleSecretName: "",
			expectError:        true,
		},
		{
			name:         "Cert Provider : InvalidProvider",
			certProvider: "InvalidProvider",
			expectError:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := tassert.New(t)
			certProviderKind = tc.certProvider
			vaultOptions.VaultToken = tc.vaultToken
			certManagerOptions.IssuerName = tc.issuerName
			caBundleSecretName = tc.caBundleSecretName
			err := validateCertificateManagerOptions()
			assert.Equal(err != nil, tc.expectError)
		})
	}
}

func TestValidateCLIParams(t *testing.T) {
	testCases := []struct {
		name                       string
		certProvider               string
		meshName                   string
		osmNamespace               string
		validatorWebhookConfigName string
		caBundleSecretName         string
		expectError                bool
	}{
		{
			name:                       "none of the necessary CLI params are empty",
			certProvider:               providers.TresorKind.String(),
			meshName:                   "test-mesh",
			osmNamespace:               "test-ns",
			validatorWebhookConfigName: "test-webhook",
			caBundleSecretName:         "test-secret",
			expectError:                false,
		},
		{
			name:                       "mesh name is empty",
			certProvider:               providers.TresorKind.String(),
			meshName:                   "",
			osmNamespace:               "test-ns",
			validatorWebhookConfigName: "test-webhook",
			caBundleSecretName:         "test-secret",
			expectError:                true,
		},
		{
			name:                       "osm namespace is empty",
			certProvider:               providers.TresorKind.String(),
			meshName:                   "test-mesh",
			osmNamespace:               "",
			validatorWebhookConfigName: "test-webhook",
			caBundleSecretName:         "test-secret",
			expectError:                true,
		},
		{
			name:                       "validator webhook is empty",
			certProvider:               providers.TresorKind.String(),
			meshName:                   "test-mesh",
			osmNamespace:               "test-ns",
			validatorWebhookConfigName: "",
			caBundleSecretName:         "test-secret",
			expectError:                true,
		},
		{
			name:                       "cabundle is empty",
			certProvider:               providers.TresorKind.String(),
			meshName:                   "test-mesh",
			osmNamespace:               "test-ns",
			validatorWebhookConfigName: "test-webhook",
			caBundleSecretName:         "",
			expectError:                true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := tassert.New(t)
			certProviderKind = tc.certProvider
			meshName = tc.meshName
			osmNamespace = tc.osmNamespace
			validatorWebhookConfigName = tc.validatorWebhookConfigName
			caBundleSecretName = tc.caBundleSecretName
			err := validateCLIParams()
			assert.Equal(err != nil, tc.expectError)
		})
	}
}
