package vault

import (
	"testing"

	tassert "github.com/stretchr/testify/assert"
)

func TestValidateOptions(t *testing.T) {
	assert := tassert.New(t)

	testCases := []struct {
		testName  string
		options   Options
		expectErr bool
	}{
		{
			testName: "invalid proto",
			options: Options{
				Protocol: "ftp",
				Host:     "vault-host",
				Token:    "vault-token",
				Role:     "vault-role",
			},
			expectErr: true,
		},
		{
			testName: "Empty host",
			options: Options{
				Protocol: "http",
				Host:     "",
				Token:    "vault-token",
				Role:     "vault-role",
			},
			expectErr: true,
		},
		{
			testName: "Empty token",
			options: Options{
				Protocol: "https",
				Host:     "vault-host",
				Token:    "",
				Role:     "vault-role",
			},
			expectErr: true,
		},
		{
			testName: "Empty role",
			options: Options{
				Protocol: "http",
				Host:     "vault-host",
				Token:    "vault-token",
				Role:     "",
			},
			expectErr: true,
		},
		{
			testName: "Empty role",
			options: Options{
				Protocol: "https",
				Host:     "vault-host",
				Token:    "vault-token",
				Role:     "",
			},
			expectErr: true,
		},
		{
			testName: "Valid config",
			options: Options{
				Protocol: "https",
				Host:     "vault-host",
				Token:    "vault-token",
				Role:     "role",
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
