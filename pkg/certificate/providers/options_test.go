package providers

import (
	"testing"

	tassert "github.com/stretchr/testify/assert"
)

func TestValidateCertManagerOptions(t *testing.T) {
	assert := tassert.New(t)

	testCases := []struct {
		testName  string
		options   CertManagerOptions
		expectErr bool
	}{
		{
			testName: "Empty issuer",
			options: CertManagerOptions{
				IssuerName:  "",
				IssuerKind:  "test-kind",
				IssuerGroup: "test-group",
			},
			expectErr: true,
		},
		{
			testName: "Empty kind",
			options: CertManagerOptions{
				IssuerName:  "test-name",
				IssuerKind:  "",
				IssuerGroup: "test-group",
			},
			expectErr: true,
		},
		{
			testName: "Empty group",
			options: CertManagerOptions{
				IssuerName:  "test-name",
				IssuerKind:  "test-kind",
				IssuerGroup: "",
			},
			expectErr: true,
		},
		{
			testName: "Valid cert manager opts",
			options: CertManagerOptions{
				IssuerName:  "test-name",
				IssuerKind:  "test-kind",
				IssuerGroup: "test-group",
			},
			expectErr: false,
		},
	}

	for _, t := range testCases {
		err := t.options.Validate()
		if t.expectErr {
			assert.Error(err, "test '%s' didn't error as expected", t.testName)
		} else {
			assert.NoError(err, "test '%s' didn't succeed as expected", t.testName)
		}
	}
}

func TestValidateVaultOptions(t *testing.T) {
	assert := tassert.New(t)

	testCases := []struct {
		testName  string
		options   VaultOptions
		expectErr bool
	}{
		{
			testName: "invalid proto",
			options: VaultOptions{
				VaultProtocol: "ftp",
				VaultHost:     "vault-host",
				VaultToken:    "vault-token",
				VaultRole:     "vault-role",
			},
			expectErr: true,
		},
		{
			testName: "Empty host",
			options: VaultOptions{
				VaultProtocol: "http",
				VaultHost:     "",
				VaultToken:    "vault-token",
				VaultRole:     "vault-role",
			},
			expectErr: true,
		},
		{
			testName: "Empty token, valid token secret",
			options: VaultOptions{
				VaultProtocol:        "https",
				VaultHost:            "vault-host",
				VaultToken:           "",
				VaultRole:            "vault-role",
				VaultTokenSecretName: "secret",
				VaultTokenSecretKey:  "key",
			},
			expectErr: false,
		},
		{
			testName: "Empty token, empty token secret",
			options: VaultOptions{
				VaultProtocol: "https",
				VaultHost:     "vault-host",
				VaultToken:    "",
				VaultRole:     "vault-role",
			},
			expectErr: true,
		},
		{
			testName: "Empty token secret key",
			options: VaultOptions{
				VaultProtocol:        "https",
				VaultHost:            "vault-host",
				VaultToken:           "",
				VaultRole:            "vault-role",
				VaultTokenSecretName: "secret",
				VaultTokenSecretKey:  "",
			},
			expectErr: true,
		},
		{
			testName: "Empty role",
			options: VaultOptions{
				VaultProtocol: "https",
				VaultHost:     "vault-host",
				VaultToken:    "vault-token",
				VaultRole:     "",
			},
			expectErr: true,
		},
		{
			testName: "Valid config",
			options: VaultOptions{
				VaultProtocol: "https",
				VaultHost:     "vault-host",
				VaultToken:    "vault-token",
				VaultRole:     "role",
			},
			expectErr: false,
		},
	}

	for _, t := range testCases {
		err := t.options.Validate()
		if t.expectErr {
			assert.Error(err, "test '%s' didn't error as expected", t.testName)
		} else {
			assert.NoError(err, "test '%s' didn't succeed as expected", t.testName)
		}
	}
}
