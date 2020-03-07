package injector

import (
	corev1 "k8s.io/api/core/v1"

	"github.com/deislabs/smc/pkg/constants"
)

// getVolumeSpec returns a list of volumes to add to the POD
func getVolumeSpec(envoyTLSSecretName string) []corev1.Volume {
	return []corev1.Volume{
		{
			// Envoy's TLS volume. This is sourced from the TLS secret
			// references by 'envoyTLSSecretName'
			Name: envoyTLSVolume,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: envoyTLSSecretName,
				},
			},
		},
		{
			Name: envoyRootCertVolume,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: constants.RootCertPemStoreName,
					},
				},
			},
		},
	}
}
