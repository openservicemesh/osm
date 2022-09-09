package injector

import (
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/pointer"

	"github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/envoy/bootstrap"
	"github.com/openservicemesh/osm/pkg/models"
	"github.com/openservicemesh/osm/pkg/utils"
)

func getPlatformSpecificSpecComponents(meshConfig v1alpha2.MeshConfig, podOS string) (podSecurityContext *corev1.SecurityContext, envoyContainer string) {
	if strings.EqualFold(podOS, constants.OSWindows) {
		podSecurityContext = &corev1.SecurityContext{
			WindowsOptions: &corev1.WindowsSecurityContextOptions{
				RunAsUserName: func() *string {
					userName := constants.EnvoyWindowsUser
					return &userName
				}(),
			},
		}
		envoyContainer = utils.GetEnvoyWindowsImage(meshConfig)
	} else {
		podSecurityContext = &corev1.SecurityContext{
			AllowPrivilegeEscalation: pointer.BoolPtr(false),
			RunAsUser: func() *int64 {
				uid := constants.EnvoyUID
				return &uid
			}(),
		}
		envoyContainer = utils.GetEnvoyImage(meshConfig)
	}
	return
}

func getEnvoySidecarContainerSpec(pod *corev1.Pod, namespace string, meshConfig v1alpha2.MeshConfig, originalHealthProbes map[string]models.HealthProbes, podOS string) corev1.Container {
	// cluster ID will be used as an identifier to the tracing sink
	// pod.Namespace is unset in the API request to the webhook so namespace is derived from req.Namespace
	clusterID := fmt.Sprintf("%s.%s", pod.Spec.ServiceAccountName, namespace)
	securityContext, containerImage := getPlatformSpecificSpecComponents(meshConfig, podOS)

	logLevel := meshConfig.Spec.Sidecar.LogLevel
	if logLevel == "" {
		logLevel = constants.DefaultEnvoyLogLevel
	}
	return corev1.Container{
		Name:            constants.EnvoyContainerName,
		Image:           containerImage,
		ImagePullPolicy: corev1.PullIfNotPresent,
		SecurityContext: securityContext,
		Ports:           getEnvoyContainerPorts(originalHealthProbes),
		VolumeMounts: []corev1.VolumeMount{{
			Name:      envoyBootstrapConfigVolume,
			ReadOnly:  true,
			MountPath: bootstrap.EnvoyProxyConfigPath,
		}},
		Command:   []string{"envoy"},
		Resources: meshConfig.Spec.Sidecar.Resources,
		Args: []string{
			"--log-level", logLevel,
			"--config-path", strings.Join([]string{bootstrap.EnvoyProxyConfigPath, bootstrap.EnvoyBootstrapConfigFile}, "/"),
			"--service-cluster", clusterID,
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
				Name: "POD_NAME",
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						FieldPath: "metadata.name",
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

func getEnvoyContainerPorts(originalHealthProbes map[string]models.HealthProbes) []corev1.ContainerPort {
	containerPorts := []corev1.ContainerPort{
		{
			Name:          constants.EnvoyAdminPortName,
			ContainerPort: constants.EnvoyAdminPort,
		},
		{
			Name:          constants.EnvoyInboundListenerPortName,
			ContainerPort: constants.EnvoyInboundListenerPort,
		},
		{
			Name:          constants.EnvoyInboundPrometheusListenerPortName,
			ContainerPort: constants.EnvoyPrometheusInboundListenerPort,
		},
	}

	var usesLiveness, usesReadiness, usesStartup bool

	for _, probes := range originalHealthProbes {
		if probes.Liveness != nil {
			usesLiveness = true
		}

		if probes.Readiness != nil {
			usesReadiness = true
		}

		if probes.Startup != nil {
			usesStartup = true
		}
	}

	if usesLiveness {
		livenessPort := corev1.ContainerPort{
			// Name must be no more than 15 characters
			Name:          "liveness-port",
			ContainerPort: constants.LivenessProbePort,
		}
		containerPorts = append(containerPorts, livenessPort)
	}

	if usesReadiness {
		readinessPort := corev1.ContainerPort{
			// Name must be no more than 15 characters
			Name:          "readiness-port",
			ContainerPort: constants.ReadinessProbePort,
		}
		containerPorts = append(containerPorts, readinessPort)
	}

	if usesStartup {
		startupPort := corev1.ContainerPort{
			// Name must be no more than 15 characters
			Name:          "startup-port",
			ContainerPort: constants.StartupProbePort,
		}
		containerPorts = append(containerPorts, startupPort)
	}

	return containerPorts
}
