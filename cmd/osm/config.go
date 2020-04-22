package main

import (
	"fmt"

	admissionv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/open-service-mesh/osm/pkg/constants"
	ns "github.com/open-service-mesh/osm/pkg/namespace"
)

const (
	defaultEnvoyImage          = "envoyproxy/envoy-alpine:v1.14.1"
	sidecarInjectorWebhookPort = 443
	defaultOSMInstanceID       = "osm-local"
)

func getCABundleSecretName() string {
	return fmt.Sprintf("osm-ca-%s", defaultOSMInstanceID)
}

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

func generateKubernetesConfig(name, namespace, serviceAccountName, containerRegistry, containerRegistrySecret string, port int32) (*appsv1.Deployment, *apiv1.Service) {
	service := &apiv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    map[string]string{"app": name},
		},
		Spec: apiv1.ServiceSpec{
			Ports: []apiv1.ServicePort{
				{
					Name: fmt.Sprintf("%s-port", name),
					Port: port,
					TargetPort: intstr.IntOrString{
						Type:   intstr.Int,
						IntVal: port,
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
			Selector: map[string]string{"app": name},
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
								"--verbosity", "trace",
								"--osmNamespace", namespace,
								"--osmID", defaultOSMInstanceID,
								"--init-container-image",
								fmt.Sprintf("%s/%s:latest", containerRegistry, "init"),
								"--sidecar-image", defaultEnvoyImage,
								"--caBundleSecretName", getCABundleSecretName(),
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
						},
					},
					ServiceAccountName: serviceAccountName,
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
func generateRBAC(namespace, serviceAccountName string) (*rbacv1.ClusterRole, *rbacv1.ClusterRoleBinding, *apiv1.ServiceAccount) {
	role := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: "osm-xds",
		},
		Rules: []rbacv1.PolicyRule{
			rbacv1.PolicyRule{
				APIGroups: []string{"apps"},
				Resources: []string{"daemonsets", "deployments", "replicasets", "statefulsets"},
				Verbs:     []string{"list", "get", "watch"},
			},
			rbacv1.PolicyRule{
				APIGroups: []string{"batch"},
				Resources: []string{"jobs"},
				Verbs:     []string{"list", "get", "watch"},
			},
			rbacv1.PolicyRule{
				APIGroups: []string{""},
				Resources: []string{"pods", "endpoints", "services", "replicationcontrollers", "namespaces"},
				Verbs:     []string{"list", "get", "watch"},
			},
			rbacv1.PolicyRule{
				APIGroups: []string{""},
				Resources: []string{"secrets", "configmaps"},
				Verbs:     []string{"create", "update"},
			},
			rbacv1.PolicyRule{
				APIGroups: []string{"split.smi-spec.io"},
				Resources: []string{"trafficsplits"},
				Verbs:     []string{"list", "get", "watch"},
			},
			rbacv1.PolicyRule{
				APIGroups: []string{"access.smi-spec.io"},
				Resources: []string{"traffictargets"},
				Verbs:     []string{"list", "get", "watch"},
			},
			rbacv1.PolicyRule{
				APIGroups: []string{"specs.smi-spec.io"},
				Resources: []string{"httproutegroups"},
				Verbs:     []string{"list", "get", "watch"},
			},
		},
	}
	roleBinding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: "osm-xds",
		},
		Subjects: []rbacv1.Subject{
			rbacv1.Subject{
				Kind:      "ServiceAccount",
				Name:      "osm-xds",
				Namespace: namespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "osm-xds",
		},
	}
	serviceAccount := &apiv1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "osm-xds",
			Namespace: namespace,
		},
	}

	return role, roleBinding, serviceAccount
}

func generateWebhookConfig(caBundle []byte, namespace string) *admissionv1beta1.MutatingWebhookConfiguration {

	webhookConfig := &admissionv1beta1.MutatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-ads-webhook", namespace),
			Namespace: namespace,
			Labels:    map[string]string{"app": "ads"},
		},
	}

	policyFail := admissionv1beta1.Fail
	path := "/mutate"
	webhooks := []admissionv1beta1.MutatingWebhook{
		{
			Name: "osm-inject.k8s.io",
			ClientConfig: admissionv1beta1.WebhookClientConfig{
				Service: &admissionv1beta1.ServiceReference{
					Namespace: namespace,
					Name:      constants.AggregatedDiscoveryServiceName,
					Path:      &path,
				},
				CABundle: caBundle,
			},
			Rules: []admissionv1beta1.RuleWithOperations{
				{
					Operations: []admissionv1beta1.OperationType{admissionv1beta1.Create},
					Rule: admissionv1beta1.Rule{
						APIGroups:   []string{""},
						APIVersions: []string{"v1"},
						Resources:   []string{"pods"},
					},
				},
			},
			FailurePolicy: &policyFail,
			NamespaceSelector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					ns.MonitorLabel: defaultOSMInstanceID,
				},
			},
		},
	}
	webhookConfig.Webhooks = webhooks
	return webhookConfig
}

func int32Ptr(i int32) *int32 { return &i }
