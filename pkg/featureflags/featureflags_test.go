package featureflags

import (
	"testing"

	tassert "github.com/stretchr/testify/assert"
)

func TestFlags(t *testing.T) {
	assert := tassert.New(t)

	// 1. Verify all optional features are disabled by default
	assert.Equal(false, IsWASMStatsEnabled())
	assert.Equal(false, IsEgressPolicyEnabled())

	// 2. Enable all optional features and verify they are enabled
	optionalFeatures := OptionalFeatures{
		WASMStats:    true,
		EgressPolicy: true,
	}
	Initialize(optionalFeatures)
	assert.Equal(true, IsWASMStatsEnabled())
	assert.Equal(true, IsEgressPolicyEnabled())

	// 3. Verify features cannot be reinitialized
	optionalFeatures = OptionalFeatures{
		WASMStats:    false,
		EgressPolicy: false,
	}
	Initialize(optionalFeatures)
	assert.Equal(true, IsWASMStatsEnabled())
	assert.Equal(true, IsEgressPolicyEnabled())
}
