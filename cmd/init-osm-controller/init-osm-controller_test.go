package main

import (
	"testing"

	tassert "github.com/stretchr/testify/assert"
)

func TestCreateDefaultMeshConfig(t *testing.T) {
	assert := tassert.New(t)

	meshConfig := createDefaultMeshConfig()
	assert.Equal(meshConfig.Spec.Traffic.EnablePermissiveTrafficPolicyMode, false)
	assert.Equal(meshConfig.Spec.Traffic.EnableEgress, false)
	assert.Equal(meshConfig.Spec.Sidecar.LogLevel, "error")
	assert.Equal(meshConfig.Spec.Sidecar.LogLevel, "error")
	assert.Equal(meshConfig.Spec.Observability.PrometheusScraping, true)
	assert.Equal(meshConfig.Spec.Observability.EnableDebugServer, false)
	assert.Equal(meshConfig.Spec.Traffic.UseHTTPSIngress, false)
	assert.Equal(meshConfig.Spec.Certificate.ServiceCertValidityDuration, "24h")
}
