package metrics

import (
	"fmt"
	"strconv"

	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	defaultPrometheusImage     = "prom/prometheus:v2.2.1"
	prometheusPort             = 7070
	prometheusScrapeAnnotation = "prometheus.io/scrape"
	prometheusPortAnnotation   = "prometheus.io/port"
)

func generatePrometheusRBAC(svc string, namespace string, serviceAccountName string) (*rbacv1.ClusterRole, *rbacv1.ClusterRoleBinding, *apiv1.ServiceAccount) {
	role := &rbacv1.ClusterRole{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ClusterRole",
			APIVersion: "rbac.authorization.k8s.io/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: svc,
		},
		Rules: []rbacv1.PolicyRule{
			rbacv1.PolicyRule{
				APIGroups: []string{""},
				Resources: []string{"nodes", "nodes/proxy", "services", "endpoints", "pods"},
				Verbs:     []string{"list", "get", "watch"},
			},
			rbacv1.PolicyRule{
				APIGroups: []string{"extensions"},
				Resources: []string{"ingresses"},
				Verbs:     []string{"list", "get", "watch"},
			},
			rbacv1.PolicyRule{
				NonResourceURLs: []string{"/metrics"},
				Verbs:           []string{"get"},
			},
		},
	}
	roleBinding := &rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ClusterRoleBinding",
			APIVersion: "rbac.authorization.k8s.io/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: svc,
		},
		Subjects: []rbacv1.Subject{
			rbacv1.Subject{
				Kind:      "ServiceAccount",
				Name:      fmt.Sprintf("%s-serviceaccount", svc),
				Namespace: namespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     svc,
		},
	}
	serviceAccount := &apiv1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ServiceAccount",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-serviceaccount", svc),
			Namespace: namespace,
		},
	}
	return role, roleBinding, serviceAccount
}

func generatePrometheusConfigMap(svc string, namespace string, prometheusYaml string) *apiv1.ConfigMap {
	configMap := &apiv1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      fmt.Sprintf("%s-server-conf", svc),
			Labels:    map[string]string{"name": fmt.Sprintf("%s-server-conf", svc)},
		},
		Data: map[string]string{"prometheus.yml": prometheusYaml},
	}
	return configMap
}

func generatePrometheusService(svc string, namespace string) *apiv1.Service {
	service := &apiv1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		Spec: apiv1.ServiceSpec{
			Selector: map[string]string{
				"app": fmt.Sprintf("%s-server", svc),
			},
			Type: apiv1.ServiceTypeNodePort,
			Ports: []apiv1.ServicePort{
				{Port: prometheusPort},
			},
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      fmt.Sprintf("%s-service", svc),
			Annotations: map[string]string{
				prometheusScrapeAnnotation: strconv.FormatBool(true),
				prometheusPortAnnotation:   strconv.Itoa(prometheusPort),
			},
		},
	}
	return service
}

func generatePrometheusDeployment(svc string, namespace string) *appsv1.Deployment {
	replicas := int32(1)
	defalutMode := int32(420)
	deployment := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-deployment", svc),
			Namespace: namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{
				"app": fmt.Sprintf("%s-server", svc),
			}},
			Strategy: appsv1.DeploymentStrategy{
				Type: appsv1.RecreateDeploymentStrategyType,
			},
			Template: apiv1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": fmt.Sprintf("%s-server", svc),
					}},
				Spec: apiv1.PodSpec{
					ServiceAccountName: fmt.Sprintf("%s-serviceaccount", svc),
					Containers: []apiv1.Container{
						{
							Name:            "prometheus",
							Image:           defaultPrometheusImage,
							ImagePullPolicy: apiv1.PullAlways,
							Args: []string{
								fmt.Sprintf("--config.file=/etc/%s/prometheus.yml", svc),
								fmt.Sprintf("--storage.tsdb.path=/%s/", svc),
								fmt.Sprintf("--web.listen-address=:%s", strconv.Itoa(prometheusPort)),
							},
							Ports: []apiv1.ContainerPort{
								{ContainerPort: prometheusPort},
							},
							VolumeMounts: []apiv1.VolumeMount{
								{Name: fmt.Sprintf("%s-config-volume", svc), MountPath: fmt.Sprintf("/etc/%s/", svc)},
								{Name: fmt.Sprintf("%s-storage-volume", svc), MountPath: fmt.Sprintf("/%s/", svc)},
							},
						},
					},
					Volumes: []apiv1.Volume{
						{
							Name: fmt.Sprintf("%s-config-volume", svc),
							VolumeSource: apiv1.VolumeSource{
								ConfigMap: &apiv1.ConfigMapVolumeSource{
									LocalObjectReference: apiv1.LocalObjectReference{
										Name: fmt.Sprintf("%s-server-conf", svc),
									},
									DefaultMode: &defalutMode,
								},
							},
						},
						{
							Name: fmt.Sprintf("%s-storage-volume", svc),
							VolumeSource: apiv1.VolumeSource{
								EmptyDir: &apiv1.EmptyDirVolumeSource{},
							},
						},
					},
				},
			},
		},
	}
	return deployment
}
