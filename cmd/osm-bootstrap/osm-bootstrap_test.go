package main

import (
	"testing"

	tassert "github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestCreateDefaultMeshConfig(t *testing.T) {
	assert := tassert.New(t)

	presetMeshConfigMap := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: presetMeshConfigName,
		},
		Data: map[string]string{
			presetMeshConfigJSONKey: `{
"sidecar": {
  "enablePrivilegedInitContainer": false,
  "logLevel": "error",
  "maxDataPlaneConnections": 0,
  "envoyImage": "envoyproxy/envoy-alpine@sha256:6502a637c6c5fba4d03d0672d878d12da4bcc7a0d0fb3f1d506982dde0039abd",
  "initContainerImage": "openservicemesh/init:v0.9.2",
  "configResyncInterval": "2s"
},
"traffic": {
	"enableEgress": true,
	"useHTTPSIngress": false,
	"enablePermissiveTrafficPolicyMode": true,
	"outboundPortExclusionList": [],
	"inboundPortExclusionList": [],
	"outboundIPRangeExclusionList": []
  },
  "observability": {
	"enableDebugServer": false,
	"osmLogLevel": "trace",
	"tracing": {
   	  "enable": false
	}
  },
  "certificate": {
	"serviceCertValidityDuration": "23h"
  },
  "featureFlags": {
	"enableWASMStats": false,
	"enableEgressPolicy": true,
	"enableMulticlusterMode": false,
	"enableAsyncProxyServiceMapping": false,
	"enableValidatingWebhook": false,
	"enableIngressBackendPolicy": true,
	"enableEnvoyActiveHealthChecks": true,
	"enableSnapshotCacheMode": true,
	"enableRetryPolicy": false
	}
}`,
		},
	}

	meshConfig := createDefaultMeshConfig(presetMeshConfigMap)
	assert.Equal(meshConfig.Name, meshConfigName)
	assert.Equal(meshConfig.Spec.Sidecar.LogLevel, "error")
	assert.Equal(meshConfig.Spec.Sidecar.ConfigResyncInterval, "2s")
	assert.False(meshConfig.Spec.Sidecar.EnablePrivilegedInitContainer)
	assert.True(meshConfig.Spec.Traffic.EnablePermissiveTrafficPolicyMode)
	assert.True(meshConfig.Spec.Traffic.EnableEgress)
	assert.False(meshConfig.Spec.FeatureFlags.EnableWASMStats)
	assert.False(meshConfig.Spec.Traffic.UseHTTPSIngress)
	assert.False(meshConfig.Spec.Observability.EnableDebugServer)
	assert.Equal(meshConfig.Spec.Certificate.ServiceCertValidityDuration, "23h")
	assert.True(meshConfig.Spec.FeatureFlags.EnableIngressBackendPolicy)
	assert.True(meshConfig.Spec.FeatureFlags.EnableEnvoyActiveHealthChecks)
	assert.False(meshConfig.Spec.FeatureFlags.EnableRetryPolicy)
}

func TestValidateCLIParams(t *testing.T) {
	assert := tassert.New(t)

	// save original global values
	prevOsmNamespace := osmNamespace

	tests := []struct {
		caseName string
		setup    func()
		verify   func(error)
	}{
		{
			caseName: "osm-namespace is empty",
			setup: func() {
				osmNamespace = ""
			},
			verify: func(err error) {
				assert.NotNil(err)
				assert.Contains(err.Error(), "--osm-namespace")
			},
		},
		{
			caseName: "osm-namespace is valid",
			setup: func() {
				osmNamespace = "valid-ns"
			},
			verify: func(err error) {
				assert.NotNil(err)
				assert.Contains(err.Error(), "--ca-bundle-secret-name")
			},
		},
		{
			caseName: "osm-namespace and ca-bundle-secret-name is valid",
			setup: func() {
				osmNamespace = "valid-ns"
				caBundleSecretName = "valid-ca-bundle"
			},
			verify: func(err error) {
				assert.Nil(err)
			},
		},
	}

	for _, tc := range tests {
		tc.setup()
		err := validateCLIParams()
		tc.verify(err)
	}

	// restore original global values
	osmNamespace = prevOsmNamespace
}
