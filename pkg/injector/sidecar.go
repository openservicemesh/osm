package injector

import (
	"strings"

	corev1 "k8s.io/api/core/v1"

	"github.com/openservicemesh/osm/pkg/constants"
)

const (
	envoyBootstrapConfigFile = "bootstrap.yaml"
	envoyProxyConfigPath     = "/etc/envoy"
	envoyContainerName       = "envoy"
)

func getEnvoySidecarContainerSpec(containerName, envoyImage, nodeID, clusterID string) []corev1.Container {
	container := corev1.Container{
		Name:            containerName,
		Image:           envoyImage,
		ImagePullPolicy: corev1.PullAlways,
		SecurityContext: &corev1.SecurityContext{
			RunAsUser: func() *int64 {
				uid := constants.EnvoyUID
				return &uid
			}(),
		},
		Ports: []corev1.ContainerPort{{
			Name:          constants.EnvoyAdminPortName,
			ContainerPort: constants.EnvoyAdminPort,
		}, {
			Name:          constants.EnvoyInboundListenerPortName,
			ContainerPort: constants.EnvoyInboundListenerPort,
		}, {
			Name:          constants.EnvoyInboundPrometheusListenerPortName,
			ContainerPort: constants.EnvoyPrometheusInboundListenerPort,
		}},
		VolumeMounts: []corev1.VolumeMount{{
			Name:      envoyBootstrapConfigVolume,
			ReadOnly:  true,
			MountPath: envoyProxyConfigPath,
		}},
		Command: []string{"envoy"},
		Args: []string{
			"--log-level", "debug", // TODO(draychev): Move to ConfigMap: Github Issue https://github.com/openservicemesh/osm/issues/1232
			"--config-path", strings.Join([]string{envoyProxyConfigPath, envoyBootstrapConfigFile}, "/"),
			"--service-node", nodeID,
			"--service-cluster", clusterID,
		},
	}

	return []corev1.Container{container}
}
