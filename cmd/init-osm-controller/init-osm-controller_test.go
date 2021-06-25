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
<<<<<<< HEAD
		Spec: v1alpha1.MeshConfigSpec{
			Sidecar: v1alpha1.SidecarSpec{
				LogLevel:                      "error",
				EnvoyImage:                    "envoyproxy/envoy-alpine:v1.18.3",
				InitContainerImage:            "openservicemesh/init:v0.9.1",
				EnablePrivilegedInitContainer: false,
				MaxDataPlaneConnections:       0,
				ConfigResyncInterval:          "2s",
			},
			Traffic: v1alpha1.TrafficSpec{
				EnableEgress:                      true,
				UseHTTPSIngress:                   false,
				EnablePermissiveTrafficPolicyMode: true,
			},
			Observability: v1alpha1.ObservabilitySpec{
				EnableDebugServer: false,
				Tracing: v1alpha1.TracingSpec{
					Enable: false,
				},
			},
			Certificate: v1alpha1.CertificateSpec{
				ServiceCertValidityDuration: "24h",
			},
=======
		Data: map[string]string{
			presetMeshConfigJSONKey: `{
"sidecar": {
  "enablePrivilegedInitContainer": false,
  "logLevel": "error",
  "maxDataPlaneConnections": 0,
  "envoyImage": "envoyproxy/envoy-alpine:v1.18.3",
  "initContainerImage": "openservicemesh/init:v0.9.0",
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
   "  enable": false
	}
  },
  "certificate": {
	"serviceCertValidityDuration": "24h"
  },
  "featureFlags": {
	"enableWASMStats": true,
	"enableEgressPolicy": true,
	"enableMulticlusterMode": false,
	"enableOSMGateway": false,
	"enableAsyncProxyServiceMapping": false,
	"enableValidatingWebhook": false
	}
}`,
>>>>>>> 2429d81d (ref(preset-mesh-config): Change kind of preset-mesh-config to ConfigMap)
		},
	}

	meshConfig := createDefaultMeshConfig(presetMeshConfigMap)
	assert.Equal(meshConfig.Name, meshConfigName)
	assert.Equal(meshConfig.Spec.Sidecar.LogLevel, "error")
	assert.Equal(meshConfig.Spec.Sidecar.ConfigResyncInterval, "2s")
	assert.Equal(meshConfig.Spec.Sidecar.EnablePrivilegedInitContainer, false)
	assert.Equal(meshConfig.Spec.Traffic.EnablePermissiveTrafficPolicyMode, true)
	assert.Equal(meshConfig.Spec.Traffic.EnableEgress, true)
	assert.Equal(meshConfig.Spec.Traffic.UseHTTPSIngress, false)
	assert.Equal(meshConfig.Spec.Observability.EnableDebugServer, false)
	assert.Equal(meshConfig.Spec.Certificate.ServiceCertValidityDuration, "24h")
}
