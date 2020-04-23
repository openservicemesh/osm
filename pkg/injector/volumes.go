package injector

import (
	corev1 "k8s.io/api/core/v1"
)

// getVolumeSpec returns a list of volumes to add to the POD
func getVolumeSpec(envoyBootstrapConfigName string) []corev1.Volume {
	return []corev1.Volume{
		{
			Name: envoyBootstrapConfigVolume,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: envoyBootstrapConfigName,
				},
			},
		},
	}
}
