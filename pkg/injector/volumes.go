package injector

import (
	corev1 "k8s.io/api/core/v1"
)

// getVolumeSpec returns a list of volumes to add to the POD
func getVolumeSpec(envoyBootstrapConfigName, envoyTLSSecretName string) []corev1.Volume {
	return []corev1.Volume{
		{
			Name: envoyBootstrapConfigVolume,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: envoyBootstrapConfigName,
					},
				},
			},
		},
		{
			// Envoy's TLS volume. This is sourced from the TLS secret
			// referenced by 'envoyTLSSecretName'
			Name: envoyTLSVolume,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: envoyTLSSecretName,
				},
			},
		},
	}
}
