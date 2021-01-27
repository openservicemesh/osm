package injector

import (
	"strings"

	corev1 "k8s.io/api/core/v1"
)

func getInitContainerSpec(containerName string, containerImage string, outboundIPRangeExclusionList []string) corev1.Container {
	iptablesInitCommandsList := generateIptablesCommands(outboundIPRangeExclusionList)
	iptablesInitCommand := strings.Join(iptablesInitCommandsList, " && ")

	return corev1.Container{
		Name:  containerName,
		Image: containerImage,
		SecurityContext: &corev1.SecurityContext{
			Capabilities: &corev1.Capabilities{
				Add: []corev1.Capability{
					"NET_ADMIN",
				},
			},
		},
		Command: []string{"/bin/sh"},
		Args: []string{
			"-c",
			iptablesInitCommand,
		},
	}
}
