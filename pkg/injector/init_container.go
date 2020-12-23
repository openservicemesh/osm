package injector

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"

	"github.com/openservicemesh/osm/pkg/constants"
)

func getInitContainerSpec(initContainer *InitContainer) (corev1.Container, error) {
	return corev1.Container{
<<<<<<< HEAD:pkg/injector/init-container.go
		Name:  data.Name,
		Image: data.Image,
		ImagePullPolicy: "Always",
=======
		Name:  initContainer.Name,
		Image: initContainer.Image,
>>>>>>> d8b189c3bbeb430f8827cd653a07b0a1fc07ae22:pkg/injector/init_container.go
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
