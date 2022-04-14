package injector

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/pointer"

	"github.com/openservicemesh/osm/pkg/configurator"
)

func getInitContainerSpec(containerName string, cfg configurator.Configurator, outboundIPRangeExclusionList []string,
	outboundIPRangeInclusionList []string, outboundPortExclusionList []int,
	inboundPortExclusionList []int, enablePrivilegedInitContainer bool, pullPolicy corev1.PullPolicy) corev1.Container {
	iptablesInitCommand := generateIptablesCommands(outboundIPRangeExclusionList, outboundIPRangeInclusionList, outboundPortExclusionList, inboundPortExclusionList)

	return corev1.Container{
		Name:            containerName,
		Image:           cfg.GetInitContainerImage(),
		ImagePullPolicy: pullPolicy,
		SecurityContext: &corev1.SecurityContext{
			Privileged: &enablePrivilegedInitContainer,
			Capabilities: &corev1.Capabilities{
				Add: []corev1.Capability{
					"NET_ADMIN",
				},
			},
			RunAsNonRoot: pointer.BoolPtr(false),
			// User ID 0 corresponds to root
			RunAsUser: pointer.Int64Ptr(0),
		},
		Command: []string{"/bin/sh"},
		Args: []string{
			"-c",
			iptablesInitCommand,
		},
		Env: []corev1.EnvVar{
			{
				Name: "POD_IP",
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						APIVersion: "v1",
						FieldPath:  "status.podIP",
					},
				},
			},
		},
	}
}
