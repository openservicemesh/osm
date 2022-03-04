package certmanager

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
			testName: "Empty issuer",
			options: Options{
				IssuerName:  "",
				IssuerKind:  "test-kind",
				IssuerGroup: "test-group",
			},
			expectErr: true,
		},
		{
			testName: "Empty kind",
			options: Options{
				IssuerName:  "test-name",
				IssuerKind:  "",
				IssuerGroup: "test-group",
			},
			expectErr: true,
		},
		{
			testName: "Empty group",
			options: Options{
				IssuerName:  "test-name",
				IssuerKind:  "test-kind",
				IssuerGroup: "",
			},
			expectErr: true,
		},
		{
			testName: "Valid cert manager opts",
			options: Options{
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
