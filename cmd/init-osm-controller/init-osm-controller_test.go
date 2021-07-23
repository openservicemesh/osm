package main

import (
	"testing"

	tassert "github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openservicemesh/osm/pkg/apis/config/v1alpha1"
)

func TestCreateDefaultMeshConfig(t *testing.T) {
	assert := tassert.New(t)

	presetMeshConfig := &v1alpha1.MeshConfig{
		TypeMeta: metav1.TypeMeta{
			Kind:       "MeshConfig",
			APIVersion: "config.openservicemesh.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: presetMeshConfigName,
		},
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
		},
	}

	meshConfig := createDefaultMeshConfig(presetMeshConfig)
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
