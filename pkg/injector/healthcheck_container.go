package injector

import (
	"os"

	corev1 "k8s.io/api/core/v1"
)

func getHealthcheckContainerSpec(containerName string) corev1.Container {
	return corev1.Container{
		Name:            containerName,
		Image:           os.Getenv("OSM_DEFAULT_HEALTHCHECK_CONTAINER_IMAGE"),
		ImagePullPolicy: corev1.PullIfNotPresent,
		Command: []string{
			"/osm-healthcheck",
		},
		Ports: []corev1.ContainerPort{
			{
				ContainerPort: healthcheckPort,
			},
		},
	}
}
