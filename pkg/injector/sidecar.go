package injector

import (
	corev1 "k8s.io/api/core/v1"

	"github.com/open-service-mesh/osm/pkg/constants"
)

const (
	envoyProxyConfigPath      = "/etc/envoy"
	envoySidecarContainerName = "envoyproxy"
	envoyTLSCertPath          = "/etc/ssl/certs"
	envoyBootstrapConfigFile  = "/etc/envoy/bootstrap.yaml"
)

func getEnvoySidecarContainerSpec(pod *corev1.Pod, data *EnvoySidecarData) (corev1.Container, error) {
	container := corev1.Container{
		Name:            data.Name,
		Image:           data.Image,
		ImagePullPolicy: corev1.PullAlways,
		SecurityContext: &corev1.SecurityContext{
			RunAsUser: func() *int64 {
				uid := constants.EnvoyUID
				return &uid
			}(),
		},
		Ports: []corev1.ContainerPort{
			{
				Name:          constants.EnvoyAdminPortName,
				ContainerPort: constants.EnvoyAdminPort,
			},
			{
				Name:          constants.EnvoyInboundListenerPortName,
				ContainerPort: constants.EnvoyInboundListenerPort,
			},
		},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      envoyBootstrapConfigVolume,
				ReadOnly:  true,
				MountPath: envoyProxyConfigPath,
			},
			{
				Name:      envoyTLSVolume,
				ReadOnly:  true,
				MountPath: envoyTLSCertPath,
			},
		},
		Command: []string{
			"envoy",
		},
		Args: []string{
			"--log-level", "debug", // TODO: remove
			"--config-path", envoyBootstrapConfigFile,
			"--service-node", data.ServiceAccount,
			"--service-cluster", data.ServiceAccount,
		},
	}

	return container, nil
}
