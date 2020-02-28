package main

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func generateCertConfig(name, namespace, key string, value []byte) *apiv1.ConfigMap {
	data := map[string]string{}
	data[key] = string(value)

	return &apiv1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: data,
	}
}

func generateNamespaceConfig(namespace string) *apiv1.Namespace {
	return &apiv1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}
}

func generateKubernetesConfig(name, namespace, containerRegistry, containerRegistrySecret string, port int32) (*appsv1.Deployment, *apiv1.Service) {
	service := &apiv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    map[string]string{"app": name},
		},
		Spec: apiv1.ServiceSpec{
			Ports: []apiv1.ServicePort{
				{
					Name: fmt.Sprintf("%s-envoy-admin-port", name),
					Port: 15000,
					TargetPort: intstr.IntOrString{
						Type:   intstr.String,
						StrVal: "admin-port",
					},
				},
				{
					Name: fmt.Sprintf("%s-port", name),
					Port: port,
					TargetPort: intstr.IntOrString{
						Type:   intstr.Int,
						IntVal: port,
					},
				},
			},
			Selector: map[string]string{"app": name},
			Type:     apiv1.ServiceTypeNodePort,
		},
	}
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    map[string]string{"app": name},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": name,
				},
			},
			Template: apiv1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": name,
					},
				},
				Spec: apiv1.PodSpec{
					Containers: []apiv1.Container{
						{
							Name:            name,
							Image:           fmt.Sprintf("%s/%s:latest", containerRegistry, name),
							ImagePullPolicy: apiv1.PullAlways,
							Ports: []apiv1.ContainerPort{
								{
									Name:          "admin-port",
									ContainerPort: 15000,
								},
								{
									Name:          fmt.Sprintf("%s-port", name),
									ContainerPort: port,
								},
							},
							Command: []string{fmt.Sprintf("/%s", name)},
							Args: []string{
								"--kubeconfig",
								"/kube/config",
								"--verbosity",
								"15",
								"--namespace",
								namespace,
								"--certpem",
								"/etc/ssl/certs/cert.pem",
								"--keypem",
								"/etc/ssl/certs/key.pem",
								"--rootcertpem",
								"/etc/ssl/certs/root-cert.pem",
							},
							Env: []apiv1.EnvVar{
								{
									Name:  "GRPC_GO_LOG_VERBOSITY_LEVEL",
									Value: "99",
								},
								{
									Name:  "GRPC_GO_LOG_VERBOSITY_LEVEL",
									Value: "info",
								},
							},

							VolumeMounts: []apiv1.VolumeMount{
								{
									Name:      "kubeconfig",
									MountPath: "/kube",
								},
								{
									Name:      fmt.Sprintf("ca-certpemstore-%s", name),
									MountPath: "/etc/ssl/certs/cert.pem",
									SubPath:   "cert.pem",
									ReadOnly:  false,
								},
								{
									Name:      fmt.Sprintf("ca-keypemstore-%s", name),
									MountPath: "/etc/ssl/certs/key.pem",
									SubPath:   "key.pem",
									ReadOnly:  false,
								},
								{
									Name:      "ca-rootcertpemstore",
									MountPath: "/etc/ssl/certs/root-cert.pem",
									SubPath:   "root-cert.pem",
									ReadOnly:  false,
								},
								{
									Name:      "ca-rootkeypemstore",
									MountPath: "/etc/ssl/certs/root-key.pem",
									SubPath:   "root-key.pem",
									ReadOnly:  false,
								},
							},
						},
					},
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
							Name: fmt.Sprintf("ca-certpemstore-%s", name),
							VolumeSource: apiv1.VolumeSource{
								ConfigMap: &apiv1.ConfigMapVolumeSource{
									LocalObjectReference: apiv1.LocalObjectReference{
										Name: fmt.Sprintf("ca-certpemstore-%s", name),
									},
								},
							},
						},
						{
							Name: "ca-rootcertpemstore",
							VolumeSource: apiv1.VolumeSource{
								ConfigMap: &apiv1.ConfigMapVolumeSource{
									LocalObjectReference: apiv1.LocalObjectReference{
										Name: "ca-rootcertpemstore",
									},
								},
							},
						},
						{
							Name: "ca-rootkeypemstore",
							VolumeSource: apiv1.VolumeSource{
								ConfigMap: &apiv1.ConfigMapVolumeSource{
									LocalObjectReference: apiv1.LocalObjectReference{
										Name: "ca-rootkeypemstore",
									},
								},
							},
						},
						{
							Name: fmt.Sprintf("ca-keypemstore-%s", name),
							VolumeSource: apiv1.VolumeSource{
								ConfigMap: &apiv1.ConfigMapVolumeSource{
									LocalObjectReference: apiv1.LocalObjectReference{
										Name: fmt.Sprintf("ca-keypemstore-%s", name),
									},
								},
							},
						},
					},
					ImagePullSecrets: []apiv1.LocalObjectReference{
						{
							Name: containerRegistrySecret,
						},
					},
				},
			},
		},
	}

	return deployment, service
}
