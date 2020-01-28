package tests

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// constant values to be used for testing
const (
	Namespace               = "--namespace--"
	Name                    = "--name--"
	Host                    = "bye.com"
	ServiceName             = "--service-name--"
	ContainerHealthPortName = "--container-health-port-name--"
	ContainerHealthPort     = int32(9090)
	ServicePort             = "service-port"
	SelectorKey             = "app"
	SelectorValue           = "frontend"
)

// NewPodFixture makes a new pod for testing
func NewPodFixture(serviceName string, ingressNamespace string, containerName string, containerPort int32) *v1.Pod {
	return &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
			Namespace: ingressNamespace,
			Labels: map[string]string{
				SelectorKey: SelectorValue,
			},
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:  serviceName,
					Image: "image",
					Ports: []v1.ContainerPort{
						{
							Name:          containerName,
							ContainerPort: containerPort,
						},
						{
							Name:          ContainerHealthPortName,
							ContainerPort: ContainerHealthPort,
						},
					},
				},
			},
		},
	}
}
