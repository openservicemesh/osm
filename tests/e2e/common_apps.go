package e2e

import (
	"context"
	"fmt"

	"k8s.io/api/admissionregistration/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// CreateServiceAccount is a wrapper to create a service account
func (td *OsmTestData) CreateServiceAccount(ns string, svcAccount *corev1.ServiceAccount) (*corev1.ServiceAccount, error) {
	svcAc, err := td.client.CoreV1().ServiceAccounts(ns).Create(context.Background(), svcAccount, metav1.CreateOptions{})
	if err != nil {
		err := fmt.Errorf("Could not create Service Account: %v", err)
		return nil, err
	}
	return svcAc, nil
}

// CreatePod is a wrapper to create a pod
func (td *OsmTestData) CreatePod(ns string, pod corev1.Pod) (*corev1.Pod, error) {
	podRet, err := td.client.CoreV1().Pods(ns).Create(context.Background(), &pod, metav1.CreateOptions{})
	if err != nil {
		err := fmt.Errorf("Could not create Pod: %v", err)
		return nil, err
	}
	return podRet, nil
}

// CreateDeployment is a wrapper to create a deployment
func (td *OsmTestData) CreateDeployment(ns string, deployment appsv1.Deployment) (*appsv1.Deployment, error) {
	deploymentRet, err := td.client.AppsV1().Deployments(ns).Create(context.Background(), &deployment, metav1.CreateOptions{})
	if err != nil {
		err := fmt.Errorf("Could not create Deployment: %v", err)
		return nil, err
	}
	return deploymentRet, nil
}

// CreateService is a wrapper to create a service
func (td *OsmTestData) CreateService(ns string, svc corev1.Service) (*corev1.Service, error) {
	sv, err := td.client.CoreV1().Services(ns).Create(context.Background(), &svc, metav1.CreateOptions{})
	if err != nil {
		err := fmt.Errorf("Could not create Service: %v", err)
		return nil, err
	}
	return sv, nil
}

// CreateMutatingWebhook is a wrapper to create a mutating webhook configuration
func (td *OsmTestData) CreateMutatingWebhook(mwhc *v1beta1.MutatingWebhookConfiguration) (*v1beta1.MutatingWebhookConfiguration, error) {
	mw, err := td.client.AdmissionregistrationV1beta1().MutatingWebhookConfigurations().Create(context.Background(), mwhc, metav1.CreateOptions{})
	if err != nil {
		err := fmt.Errorf("Could not create MutatingWebhook: %v", err)
		return nil, err
	}
	return mw, nil
}

// GetMutatingWebhook is a wrapper to get a mutating webhook configuration
func (td *OsmTestData) GetMutatingWebhook(mwhcName string) (*v1beta1.MutatingWebhookConfiguration, error) {
	mwhc, err := td.client.AdmissionregistrationV1beta1().MutatingWebhookConfigurations().Get(context.Background(), mwhcName, metav1.GetOptions{})
	if err != nil {
		err := fmt.Errorf("Could not get MutatingWebhook: %v", err)
		return nil, err
	}
	return mwhc, nil
}

/* Application templates
 * The following functions contain high level helpers to create and get test application definitions.
 *
 * These abstractions aim to simplify and avoid tests having to individually type the the same k8s definitions for
 * some common or recurrent deployment forms.
 */

// SimplePodAppDef defines some parametrization to create a pod-based application from template
type SimplePodAppDef struct {
	namespace string
	name      string
	image     string
	command   []string
	args      []string
	ports     []int
}

// SimplePodApp creates returns a set of k8s typed definitions for a pod-based k8s definition.
// Includes Pod, Service and ServiceAccount types
func (td *OsmTestData) SimplePodApp(def SimplePodAppDef) (corev1.ServiceAccount, corev1.Pod, corev1.Service) {
	serviceAccountDefinition := corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name: def.name,
		},
	}

	podDefinition := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: def.name,
			Labels: map[string]string{
				"app": def.name,
			},
		},
		Spec: corev1.PodSpec{
			ServiceAccountName: def.name,
			Containers: []corev1.Container{
				{
					Name:            def.name,
					Image:           def.image,
					ImagePullPolicy: corev1.PullIfNotPresent,
					Resources: corev1.ResourceRequirements{
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("100m"),
							corev1.ResourceMemory: resource.MustParse("128Mi"),
						},
					},
				},
			},
		},
	}

	if td.AreRegistryCredsPresent() {
		podDefinition.Spec.ImagePullSecrets = []corev1.LocalObjectReference{
			{
				Name: registrySecretName,
			},
		}
	}
	if def.command != nil && len(def.command) > 0 {
		podDefinition.Spec.Containers[0].Command = def.command
	}
	if def.args != nil && len(def.args) > 0 {
		podDefinition.Spec.Containers[0].Args = def.args
	}

	serviceDefinition := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: def.name,
			Labels: map[string]string{
				"app": def.name,
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": def.name,
			},
		},
	}

	if def.ports != nil && len(def.ports) > 0 {
		podDefinition.Spec.Containers[0].Ports = []corev1.ContainerPort{}
		serviceDefinition.Spec.Ports = []corev1.ServicePort{}

		for _, p := range def.ports {
			podDefinition.Spec.Containers[0].Ports = append(podDefinition.Spec.Containers[0].Ports,
				corev1.ContainerPort{
					ContainerPort: int32(p),
				},
			)
			serviceDefinition.Spec.Ports = append(serviceDefinition.Spec.Ports, corev1.ServicePort{
				Port:       int32(p),
				TargetPort: intstr.FromInt(p),
			})
		}
	}

	return serviceAccountDefinition, podDefinition, serviceDefinition
}

// SimpleDeploymentAppDef defines some parametrization to create a deployment-based application from template
type SimpleDeploymentAppDef struct {
	namespace    string
	name         string
	image        string
	replicaCount int32
	command      []string
	args         []string
	ports        []int
}

// SimpleDeploymentApp creates returns a set of k8s typed definitions for a deployment-based k8s definition.
// Includes Deployment, Service and ServiceAccount types
func (td *OsmTestData) SimpleDeploymentApp(def SimpleDeploymentAppDef) (corev1.ServiceAccount, appsv1.Deployment, corev1.Service) {
	serviceAccountDefinition := corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      def.name,
			Namespace: def.namespace,
		},
	}

	// Required, as replica count is a pointer to distinguish between 0 and not specified
	replicaCountExplicitDeclaration := def.replicaCount

	deploymentDefinition := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      def.name,
			Namespace: def.namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicaCountExplicitDeclaration,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": def.name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": def.name,
					},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: def.name,
					Containers: []corev1.Container{
						{
							Name:            def.name,
							Image:           def.image,
							ImagePullPolicy: corev1.PullIfNotPresent,
						},
					},
				},
			},
		},
	}

	if td.AreRegistryCredsPresent() {
		deploymentDefinition.Spec.Template.Spec.ImagePullSecrets = []corev1.LocalObjectReference{
			{
				Name: registrySecretName,
			},
		}
	}

	if def.command != nil && len(def.command) > 0 {
		deploymentDefinition.Spec.Template.Spec.Containers[0].Command = def.command
	}
	if def.args != nil && len(def.args) > 0 {
		deploymentDefinition.Spec.Template.Spec.Containers[0].Args = def.args
	}

	serviceDefinition := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      def.name,
			Namespace: def.namespace,
			Labels: map[string]string{
				"app": def.name,
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": def.name,
			},
		},
	}

	if def.ports != nil && len(def.ports) > 0 {
		deploymentDefinition.Spec.Template.Spec.Containers[0].Ports = []corev1.ContainerPort{}
		serviceDefinition.Spec.Ports = []corev1.ServicePort{}

		for _, p := range def.ports {
			deploymentDefinition.Spec.Template.Spec.Containers[0].Ports = append(deploymentDefinition.Spec.Template.Spec.Containers[0].Ports,
				corev1.ContainerPort{
					ContainerPort: int32(p),
				},
			)

			serviceDefinition.Spec.Ports = append(serviceDefinition.Spec.Ports, corev1.ServicePort{
				Port:       int32(p),
				TargetPort: intstr.FromInt(p),
			})
		}
	}

	return serviceAccountDefinition, deploymentDefinition, serviceDefinition
}
