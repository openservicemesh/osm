package main

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func generateNamespaceConfig(namespace string) *apiv1.Namespace {
	return &apiv1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}
}

func generateRDSKubernetesConfig(namespace, containerRegistry, containerRegistrySecret string) (*appsv1.Deployment, *apiv1.Service) {
	service := &apiv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rds",
			Namespace: namespace,
			Labels:    map[string]string{"app": "rds"},
		},
		Spec: apiv1.ServiceSpec{
			Ports: []apiv1.ServicePort{
				{
					Name: "rds-envoy-admin-port",
					Port: 15000,
					TargetPort: intstr.IntOrString{
						Type:   intstr.String,
						StrVal: "admin-port",
					},
				},
				{
					Name: "rds-port",
					Port: 15126,
					TargetPort: intstr.IntOrString{
						Type:   intstr.Int,
						IntVal: 15126,
					},
				},
			},
			Selector: map[string]string{"app": "rds"},
			Type:     apiv1.ServiceTypeNodePort,
		},
	}
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rds",
			Namespace: namespace,
			Labels:    map[string]string{"app": "rds"},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "rds",
				},
			},
			Template: apiv1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "rds",
					},
				},
				Spec: apiv1.PodSpec{
					Containers: []apiv1.Container{
						{
							Name:            "rds",
							Image:           fmt.Sprintf("%s/rds:latest", containerRegistry),
							ImagePullPolicy: apiv1.PullAlways,
							Ports: []apiv1.ContainerPort{
								{
									Name:          "admin-port",
									ContainerPort: 15000,
								},
								{
									Name:          "rds-port",
									ContainerPort: 15126,
								},
							},
							Command: []string{"/rds"},
							Args: []string{
								"--kubeconfig",
								"/kube/config",
								"--verbosity",
								"15",
								"--namespace",
								"smc",
								"--certpem",
								"/etc/ssl/certs/cert.pem",
								"--keypem",
								"/etc/ssl/certs/key.pem",
								"--rootcertpem",
								"--/etc/ssl/certs/cert.pem",
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
									Name:      "ca-certpemstore",
									MountPath: "/etc/ssl/certs/cert.pem",
									SubPath:   "cert.pem",
									ReadOnly:  false,
								},
								{
									Name:      "ca-keypemstore",
									MountPath: "/etc/ssl/certs/key.pem",
									SubPath:   "key.pem",
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
							Name: "ca-certpemstore",
							VolumeSource: apiv1.VolumeSource{
								ConfigMap: &apiv1.ConfigMapVolumeSource{
									LocalObjectReference: apiv1.LocalObjectReference{
										Name: "ca-certpemstore",
									},
								},
							},
						},
						{
							Name: "ca-keypemstore",
							VolumeSource: apiv1.VolumeSource{
								ConfigMap: &apiv1.ConfigMapVolumeSource{
									LocalObjectReference: apiv1.LocalObjectReference{
										Name: "ca-keypemstore",
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

func generateEDSKubernetesConfig(namespace, containerRegistry, containerRegistrySecret string) (*appsv1.Deployment, *apiv1.Service) {
	service := &apiv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "eds",
			Namespace: namespace,
			Labels:    map[string]string{"app": "eds"},
		},
		Spec: apiv1.ServiceSpec{
			Ports: []apiv1.ServicePort{
				{
					Name: "eds-envoy-admin-port",
					Port: 15000,
					TargetPort: intstr.IntOrString{
						Type:   intstr.String,
						StrVal: "admin-port",
					},
				},
				{
					Name: "eds-port",
					Port: 15124,
					TargetPort: intstr.IntOrString{
						Type:   intstr.Int,
						IntVal: 15124,
					},
				},
			},
			Selector: map[string]string{"app": "eds"},
			Type:     apiv1.ServiceTypeNodePort,
		},
	}
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "eds",
			Namespace: namespace,
			Labels:    map[string]string{"app": "eds"},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "eds",
				},
			},
			Template: apiv1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "eds",
					},
				},
				Spec: apiv1.PodSpec{
					Containers: []apiv1.Container{
						{
							Name:            "eds",
							Image:           fmt.Sprintf("%s/eds:latest", containerRegistry),
							ImagePullPolicy: apiv1.PullAlways,
							Ports: []apiv1.ContainerPort{
								{
									Name:          "admin-port",
									ContainerPort: 15000,
								},
								{
									Name:          "eds-port",
									ContainerPort: 15124,
								},
							},
							Command: []string{"/eds"},
							Args: []string{
								"--kubeconfig",
								"/kube/config",
								"--verbosity",
								"15",
								"smc",
								"--certpem",
								"/etc/ssl/certs/cert.pem",
								"--keypem",
								"/etc/ssl/certs/key.pem",
								"--rootcertpem",
								"--/etc/ssl/certs/cert.pem",
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
									Name:      "ca-certpemstore",
									MountPath: "/etc/ssl/certs/cert.pem",
									SubPath:   "cert.pem",
									ReadOnly:  false,
								},
								{
									Name:      "ca-keypemstore",
									MountPath: "/etc/ssl/certs/key.pem",
									SubPath:   "key.pem",
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
							Name: "ca-certpemstore",
							VolumeSource: apiv1.VolumeSource{
								ConfigMap: &apiv1.ConfigMapVolumeSource{
									LocalObjectReference: apiv1.LocalObjectReference{
										Name: "ca-certpemstore",
									},
								},
							},
						},
						{
							Name: "ca-keypemstore",
							VolumeSource: apiv1.VolumeSource{
								ConfigMap: &apiv1.ConfigMapVolumeSource{
									LocalObjectReference: apiv1.LocalObjectReference{
										Name: "ca-keypemstore",
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

func generateCDSKubernetesConfig(namespace, containerRegistry, containerRegistrySecret string) (*appsv1.Deployment, *apiv1.Service) {

	service := &apiv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cds",
			Namespace: namespace,
			Labels:    map[string]string{"app": "cds"},
		},
		Spec: apiv1.ServiceSpec{
			Ports: []apiv1.ServicePort{
				{
					Name: "cds-envoy-admin-port",
					Port: 15000,
					TargetPort: intstr.IntOrString{
						Type:   intstr.String,
						StrVal: "admin-port",
					},
				},
				{
					Name: "cds-port",
					Port: 15124,
					TargetPort: intstr.IntOrString{
						Type:   intstr.Int,
						IntVal: 15124,
					},
				},
			},
			Selector: map[string]string{"app": "cds"},
			Type:     apiv1.ServiceTypeNodePort,
		},
	}

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cds",
			Namespace: namespace,
			Labels:    map[string]string{"app": "cds"},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "cds",
				},
			},
			Template: apiv1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "cds",
					},
				},
				Spec: apiv1.PodSpec{
					Containers: []apiv1.Container{
						{
							Name:            "cds", //TODO was named curl mistake??
							Image:           fmt.Sprintf("%s/cds:latest", containerRegistry),
							ImagePullPolicy: apiv1.PullAlways,
							Ports: []apiv1.ContainerPort{
								{
									Name:          "admin-port",
									ContainerPort: 15000,
								},
								{
									Name:          "cds-port",
									ContainerPort: 15125,
								},
							},
							Command: []string{"/cds"},
							Args: []string{
								"--kubeconfig",
								"/kube/config",
								"--verbosity",
								"15",
								"--namespace",
								"smc",
								"--certpem",
								"/etc/ssl/certs/cert.pem",
								"--keypem",
								"/etc/ssl/certs/key.pem",
								"--rootcertpem",
								"--/etc/ssl/certs/cert.pem",
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
									Name:      "ca-certpemstore",
									MountPath: "/etc/ssl/certs/cert.pem",
									SubPath:   "cert.pem",
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
							Name: "ca-certpemstore",
							VolumeSource: apiv1.VolumeSource{
								ConfigMap: &apiv1.ConfigMapVolumeSource{
									LocalObjectReference: apiv1.LocalObjectReference{
										Name: "ca-certpemstore",
									},
								},
							},
						},
						{
							Name: "ca-keypemstore",
							VolumeSource: apiv1.VolumeSource{
								ConfigMap: &apiv1.ConfigMapVolumeSource{
									LocalObjectReference: apiv1.LocalObjectReference{
										Name: "ca-keypemstore",
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

func generateSDSKubernetesConfig(namespace, containerRegistry, containerRegistrySecret string) (*appsv1.Deployment, *apiv1.Service) {
	service := &apiv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: "sds",
		},
		Spec: apiv1.ServiceSpec{
			Ports: []apiv1.ServicePort{
				{
					Name: "admin-port",
					Port: 15000,
					TargetPort: intstr.IntOrString{
						Type:   intstr.String,
						StrVal: "admin-port",
					}, //TODO: double check these ports
				},
				{
					Name: "sds-port",
					Port: 15123,
					TargetPort: intstr.IntOrString{
						Type:   intstr.Int,
						IntVal: 15123,
					}, //TODO: double check these ports
				},
			},
			Selector: map[string]string{"app": "sds"},
			Type:     apiv1.ServiceTypeNodePort,
		},
	}

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "sds",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "sds",
				},
			},
			Template: apiv1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "sds",
					},
				},
				Spec: apiv1.PodSpec{
					Containers: []apiv1.Container{
						{
							Name:            "sds",
							Image:           fmt.Sprintf("%s/sds:latest", containerRegistry),
							ImagePullPolicy: apiv1.PullAlways,
							Ports: []apiv1.ContainerPort{
								{
									Name:          "admin-port",
									ContainerPort: 15000,
								},
								{
									Name:          "sds-port",
									ContainerPort: 15123,
								},
							},
							Command: []string{"/sds"},
							Args: []string{
								"--kubeconfig",
								"/kube/config",
								"--verbosity",
								"15",
								"--certpem",
								"/etc/ssl/certs/cert.pem",
								"--keypem",
								"/etc/ssl/certs/key.pem",
								"--rootcertpem",
								"/etc/ssl/certs/cert.pem",
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
