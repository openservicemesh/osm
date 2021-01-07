package utils

import (
	"os"
	"testing"

	tassert "github.com/stretchr/testify/assert"
)

func TestGetEnv(t *testing.T) {
	const (
		EnvVarName         = "TEST_VAR"
		EnvVarValue        = "test_value"
		EnvVarDefaultValue = "default_value"
	)

	assert := tassert.New(t)

	// make sure the variable is unset before starting the test
	assert.NoError(os.Unsetenv(EnvVarName))

	// Expect Default when not set
	assert.Equal(EnvVarDefaultValue, GetEnv(EnvVarName, EnvVarDefaultValue))

	// Set it, expect actual value when set
	assert.NoError(os.Setenv(EnvVarName, EnvVarValue))
	assert.Equal(EnvVarValue, GetEnv(EnvVarName, EnvVarDefaultValue))

	// Unset it, expect default again
	assert.NoError(os.Unsetenv(EnvVarName))
	assert.Equal(EnvVarDefaultValue, GetEnv(EnvVarName, EnvVarDefaultValue))
}
