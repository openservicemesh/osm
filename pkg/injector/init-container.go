package injector

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"

	"github.com/openservicemesh/osm/pkg/constants"
)

const (
	// InitContainerName is the name of the init container
	InitContainerName = "osm-init"
)

func getInitContainerSpec(pod *corev1.Pod, data *InitContainerData) (corev1.Container, error) {
	return corev1.Container{
		Name:  data.Name,
		Image: data.Image,
		ImagePullPolicy: "Always",
		SecurityContext: &corev1.SecurityContext{
			Capabilities: &corev1.Capabilities{
				Add: []corev1.Capability{
					"NET_ADMIN",
				},
			},
		},
		Env: []corev1.EnvVar{
			{
				Name:  "OSM_PROXY_UID",
				Value: fmt.Sprintf("%d", constants.EnvoyUID),
			},
			{
				Name:  "OSM_ENVOY_INBOUND_PORT",
				Value: fmt.Sprintf("%d", constants.EnvoyInboundListenerPort),
			},
			{
				Name:  "OSM_ENVOY_OUTBOUND_PORT",
				Value: fmt.Sprintf("%d", constants.EnvoyOutboundListenerPort),
			},
		},
	}, nil
}
