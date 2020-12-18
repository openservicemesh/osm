package injector

import (
	"strings"

	corev1 "k8s.io/api/core/v1"

	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/envoy"
)

const (
	envoyBootstrapConfigFile = "bootstrap.yaml"
	envoyProxyConfigPath     = "/etc/envoy"
)

func getEnvoySidecarContainerSpec(containerName, envoyImage, nodeID, clusterID string, cfg configurator.Configurator) corev1.Container {
	return corev1.Container{
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
			"--log-level", cfg.GetEnvoyLogLevel(),
			"--config-path", strings.Join([]string{envoyProxyConfigPath, envoyBootstrapConfigFile}, "/"),
			"--service-node", envoy.GetEnvoyServiceNodeID(nodeID),
			"--service-cluster", clusterID,
			"--bootstrap-version 3",
		},
		Env: []corev1.EnvVar{
			{
				Name: "POD_UID",
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						FieldPath: "metadata.uid",
					},
				},
			},
			{
				Name: "POD_NAMESPACE",
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						FieldPath: "metadata.namespace",
					},
				},
			},
			{
				Name: "POD_IP",
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						FieldPath: "status.podIP",
					},
				},
			},
			{
				Name: "SERVICE_ACCOUNT",
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						FieldPath: "spec.serviceAccountName",
					},
				},
			},
		},
	}
}
