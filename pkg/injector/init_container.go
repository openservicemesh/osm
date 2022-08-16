package injector

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/pointer"

	"github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"
	"github.com/openservicemesh/osm/pkg/utils"
)

func getInitContainerSpec(containerName string, meshConfig v1alpha2.MeshConfig, outboundIPRangeExclusionList []string,
	outboundIPRangeInclusionList []string, outboundPortExclusionList []int,
	inboundPortExclusionList []int, enablePrivilegedInitContainer bool, pullPolicy corev1.PullPolicy, networkInterfaceExclusionList []string) corev1.Container {
	proxyMode := meshConfig.Spec.Sidecar.LocalProxyMode
	iptablesInitCommand := generateIptablesCommands(proxyMode, outboundIPRangeExclusionList, outboundIPRangeInclusionList, outboundPortExclusionList, inboundPortExclusionList, networkInterfaceExclusionList)

	return corev1.Container{
		Name:            containerName,
		Image:           utils.GetInitContainerImage(meshConfig),
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
