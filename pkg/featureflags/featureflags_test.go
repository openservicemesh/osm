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
	assert.Equal(false, IsMulticlusterModeEnabled())

	// 2. Enable all optional features and verify they are enabled
	optionalFeatures := OptionalFeatures{
		WASMStats:        true,
		EgressPolicy:     true,
		MulticlusterMode: true,
	}
	Initialize(optionalFeatures)
	assert.Equal(true, IsWASMStatsEnabled())
	assert.Equal(true, IsEgressPolicyEnabled())
	assert.Equal(true, IsMulticlusterModeEnabled())

	// 3. Verify features cannot be reinitialized
	optionalFeatures = OptionalFeatures{
		WASMStats:        false,
		EgressPolicy:     false,
		MulticlusterMode: false,
	}
	Initialize(optionalFeatures)
	assert.Equal(true, IsWASMStatsEnabled())
	assert.Equal(true, IsEgressPolicyEnabled())
	assert.Equal(true, IsMulticlusterModeEnabled())
}
