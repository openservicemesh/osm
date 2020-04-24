package xds

import (
	"fmt"
	"os"
	"path"

	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/open-service-mesh/osm/demo/cmd/common"
	"github.com/open-service-mesh/osm/pkg/constants"
)

const (
	defaultEnvoyImage          = "envoyproxy/envoy-alpine:v1.14.1"
	sidecarInjectorWebhookPort = 443
)

func getXDSLabelMeta(namespace string) metav1.ObjectMeta {
	labels := map[string]string{
		"app": constants.AggregatedDiscoveryServiceName,
	}

	meta := metav1.ObjectMeta{
		Name:      constants.AggregatedDiscoveryServiceName,
		Namespace: namespace,
		Labels:    labels,
	}
	return meta
}

func generateXDSService(namespace string) *apiv1.Service {
	meta := getXDSLabelMeta(namespace)
	service := &apiv1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: meta,
		Spec: apiv1.ServiceSpec{
			Ports: []apiv1.ServicePort{
				{
					Name: fmt.Sprintf("%s-port", constants.AggregatedDiscoveryServiceName),
					Port: constants.AggregatedDiscoveryServicePort,
					TargetPort: intstr.IntOrString{
						IntVal: constants.AggregatedDiscoveryServicePort,
					},
				},
				{
					Name: "sidecar-injector",
					Port: sidecarInjectorWebhookPort,
					TargetPort: intstr.IntOrString{
						IntVal: constants.InjectorWebhookPort,
					},
				},
			},
			Selector: map[string]string{
				"app": constants.AggregatedDiscoveryServiceName,
			},
		},
	}
	return service
}

func generateXDSPod(namespace string) *apiv1.Pod {
	acr := os.Getenv(common.ContainerRegistryEnvVar)
	adsVersion := os.Getenv(common.ContainerTag)
	containerRegistryCredsName := os.Getenv(common.ContainerRegistryCredsEnvVar)
	azureSubscription := os.Getenv(common.AzureSubscription)
	initContainer := path.Join(acr, "init")
	osmID := os.Getenv(common.OsmIDEnvVar)

	meta := getXDSLabelMeta(namespace)
	args := []string{
		"--kubeconfig", "/kube/config",
		"--azureSubscriptionID", azureSubscription,
		"--verbosity", "trace",
		"--osmID", osmID,
		"--osmNamespace", namespace,
		"--init-container-image", initContainer,
		"--sidecar-image", defaultEnvoyImage,
	}

	if os.Getenv(common.IsGithubEnvVar) != "true" {
		args = append([]string{
			"--azureAuthFile", "/azure/azureAuth.json",
		}, args...)
	}

	pod := &apiv1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "",
			APIVersion: "",
		},
		ObjectMeta: meta,
		Spec: apiv1.PodSpec{
			Volumes: []apiv1.Volume{
				{
					Name: "kubeconfig",
					VolumeSource: apiv1.VolumeSource{
						ConfigMap: &apiv1.ConfigMapVolumeSource{
							LocalObjectReference: apiv1.LocalObjectReference{
								Name: "kubeconfig",
							},
						},
					},
				},
				{
					Name: "azureconfig",
					VolumeSource: apiv1.VolumeSource{
						ConfigMap: &apiv1.ConfigMapVolumeSource{
							LocalObjectReference: apiv1.LocalObjectReference{
								Name: "azureconfig",
							},
						},
					},
				},

				{
					Name: "webhook-tls-certs",
					VolumeSource: apiv1.VolumeSource{
						Secret: &apiv1.SecretVolumeSource{
							SecretName: "webhook-tls-certs",
						},
					},
				},
			},
			ImagePullSecrets: []apiv1.LocalObjectReference{
				{
					Name: containerRegistryCredsName,
				},
			},
			InitContainers: nil,
			Containers: []apiv1.Container{
				{
					Image:           fmt.Sprintf("%s/%s:%s", acr, constants.AggregatedDiscoveryServiceName, adsVersion),
					ImagePullPolicy: apiv1.PullAlways,
					Name:            constants.AggregatedDiscoveryServiceName,
					Ports: []apiv1.ContainerPort{
						{
							ContainerPort: constants.AggregatedDiscoveryServicePort,
							Name:          fmt.Sprintf("%s-port", constants.AggregatedDiscoveryServiceName),
						},
					},
					Command: []string{
						"/ads",
					},
					Env: []apiv1.EnvVar{{
						Name:  common.HumanReadableLogMessagesEnvVar,
						Value: os.Getenv(common.HumanReadableLogMessagesEnvVar),
					}},
					Args: args,
					VolumeMounts: []apiv1.VolumeMount{
						{
							Name:      "kubeconfig",
							MountPath: "/kube",
						},

						{
							Name:      "azureconfig",
							MountPath: "/azure",
						},

						{
							Name:      "webhook-tls-certs",
							MountPath: "/run/secrets/tls",
							ReadOnly:  true,
						},
					},
					// ReadinessProbe
					// LivenessProbe
				},
			},
		},
	}
	return pod
}
