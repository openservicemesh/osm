package injector

import (
	corev1 "k8s.io/api/core/v1"

	"github.com/deislabs/smc/pkg/constants"
)

const (
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
				Name:      envoyTLSVolume,
				ReadOnly:  true,
				MountPath: envoyTLSCertPath,
			},
			{
				Name:      envoyRootCertVolume,
				ReadOnly:  true,
				MountPath: constants.RootCertPath,
			},
		},
		Command: []string{
			"envoy",
		},
		Args: []string{
			"--log-level", "debug", // TODO: remove
			"--config-path", envoyBootstrapConfigFile,
			"--service-node", data.Service,
			"--service-cluster", data.Service,
		},
	}

	return container, nil
}
