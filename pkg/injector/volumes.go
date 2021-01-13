package injector

import (
	corev1 "k8s.io/api/core/v1"

	"github.com/openservicemesh/osm/pkg/featureflags"
)

// getVolumeSpec returns a list of volumes to add to the POD
func getVolumeSpec(envoyBootstrapConfigName string) []corev1.Volume {
	volumes := []corev1.Volume{
		{
			Name: envoyBootstrapConfigVolume,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: envoyBootstrapConfigName,
				},
			},
		},
	}

	if featureflags.IsWASMStatsEnabled() {
		volumes = append(volumes, corev1.Volume{
			Name: envoyStatsWASMVolumeName,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: statsWASMConfigMapName,
					},
				},
			},
		})
	}

	return volumes
}
