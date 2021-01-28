package framework

import (
	"context"
	"fmt"

	goversion "github.com/hashicorp/go-version"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"k8s.io/api/admissionregistration/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"

	k8s "github.com/openservicemesh/osm/pkg/kubernetes"
)

const (
	// DefaultOsmGrafanaPort is the default Grafana port
	DefaultOsmGrafanaPort = 3000
	// DefaultOsmPrometheusPort default OSM prometheus port
	DefaultOsmPrometheusPort = 7070

	// OsmControllerAppLabel is the OSM Controller deployment app label
	OsmControllerAppLabel = "osm-controller"
	// OsmGrafanaAppLabel is the OSM Grafana deployment app label
	OsmGrafanaAppLabel = "osm-grafana"
	// OsmPrometheusAppLabel is the OSM Prometheus deployment app label
	OsmPrometheusAppLabel = "osm-prometheus"

	// OSM Grafana Dashboard specifics

	// MeshDetails is dashboard uuid and name as we have them load in Grafana
	MeshDetails string = "PLyKJcHGz/mesh-and-envoy-details"

	// MemRSSPanel is the ID of the MemRSS panel on OSM's MeshDetails dashboard
	MemRSSPanel int = 13

	// CPUPanel is the ID of the CPU panel on OSM's MeshDetails dashboard
	CPUPanel int = 14

	// AppProtocolHTTP is the HTTP application protocol
	AppProtocolHTTP = "http"

	// AppProtocolTCP is the TCP application protocol
	AppProtocolTCP = "tcp"
)

// CreateServiceAccount is a wrapper to create a service account
func (td *OsmTestData) CreateServiceAccount(ns string, svcAccount *corev1.ServiceAccount) (*corev1.ServiceAccount, error) {
	svcAc, err := td.Client.CoreV1().ServiceAccounts(ns).Create(context.Background(), svcAccount, metav1.CreateOptions{})
	if err != nil {
		err := fmt.Errorf("Could not create Service Account: %v", err)
		return nil, err
	}
	return svcAc, nil
}

// CreatePod is a wrapper to create a pod
func (td *OsmTestData) CreatePod(ns string, pod corev1.Pod) (*corev1.Pod, error) {
	podRet, err := td.Client.CoreV1().Pods(ns).Create(context.Background(), &pod, metav1.CreateOptions{})
	if err != nil {
		err := fmt.Errorf("Could not create Pod: %v", err)
		return nil, err
	}
	return podRet, nil
}

// CreateDeployment is a wrapper to create a deployment
func (td *OsmTestData) CreateDeployment(ns string, deployment appsv1.Deployment) (*appsv1.Deployment, error) {
	deploymentRet, err := td.Client.AppsV1().Deployments(ns).Create(context.Background(), &deployment, metav1.CreateOptions{})
	if err != nil {
		err := fmt.Errorf("Could not create Deployment: %v", err)
		return nil, err
	}
	return deploymentRet, nil
}

// CreateService is a wrapper to create a service
func (td *OsmTestData) CreateService(ns string, svc corev1.Service) (*corev1.Service, error) {
	sv, err := td.Client.CoreV1().Services(ns).Create(context.Background(), &svc, metav1.CreateOptions{})
	if err != nil {
		err := fmt.Errorf("Could not create Service: %v", err)
		return nil, err
	}
	return sv, nil
}

// CreateMutatingWebhook is a wrapper to create a mutating webhook configuration
func (td *OsmTestData) CreateMutatingWebhook(mwhc *v1beta1.MutatingWebhookConfiguration) (*v1beta1.MutatingWebhookConfiguration, error) {
	mw, err := td.Client.AdmissionregistrationV1beta1().MutatingWebhookConfigurations().Create(context.Background(), mwhc, metav1.CreateOptions{})
	if err != nil {
		err := fmt.Errorf("Could not create MutatingWebhook: %v", err)
		return nil, err
	}
	return mw, nil
}

// GetMutatingWebhook is a wrapper to get a mutating webhook configuration
func (td *OsmTestData) GetMutatingWebhook(mwhcName string) (*v1beta1.MutatingWebhookConfiguration, error) {
	mwhc, err := td.Client.AdmissionregistrationV1beta1().MutatingWebhookConfigurations().Get(context.Background(), mwhcName, metav1.GetOptions{})
	if err != nil {
		err := fmt.Errorf("Could not get MutatingWebhook: %v", err)
		return nil, err
	}
	return mwhc, nil
}

// GetPodsForLabel returns the Pods matching a specific `appLabel`
func (td *OsmTestData) GetPodsForLabel(ns string, labelSel metav1.LabelSelector) ([]corev1.Pod, error) {
	// Apparently there has to be a conversion between metav1 and labels
	labelMap, _ := metav1.LabelSelectorAsMap(&labelSel)

	pods, err := Td.Client.CoreV1().Pods(ns).List(context.Background(), metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(labelMap).String(),
	})

	if err != nil {
		return nil, err
	}

	return pods.Items, nil
}

/* Application templates
 * The following functions contain high level helpers to create and get test application definitions.
 *
 * These abstractions aim to simplify and avoid tests having to individually type the the same k8s definitions for
 * some common or recurrent deployment forms.
 */

// SimplePodAppDef defines some parametrization to create a pod-based application from template
type SimplePodAppDef struct {
	Namespace   string
	Name        string
	Image       string
	Command     []string
	Args        []string
	Ports       []int
	AppProtocol string
}

// SimplePodApp creates returns a set of k8s typed definitions for a pod-based k8s definition.
// Includes Pod, Service and ServiceAccount types
func (td *OsmTestData) SimplePodApp(def SimplePodAppDef) (corev1.ServiceAccount, corev1.Pod, corev1.Service) {
	serviceAccountDefinition := corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name: def.Name,
		},
	}

	podDefinition := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: def.Name,
			Labels: map[string]string{
				"app": def.Name,
			},
		},
		Spec: corev1.PodSpec{
			ServiceAccountName: def.Name,
			Containers: []corev1.Container{
				{
					Name:            def.Name,
					Image:           def.Image,
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
	if def.Command != nil && len(def.Command) > 0 {
		podDefinition.Spec.Containers[0].Command = def.Command
	}
	if def.Args != nil && len(def.Args) > 0 {
		podDefinition.Spec.Containers[0].Args = def.Args
	}

	serviceDefinition := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: def.Name,
			Labels: map[string]string{
				"app": def.Name,
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": def.Name,
			},
		},
	}

	if def.Ports != nil && len(def.Ports) > 0 {
		podDefinition.Spec.Containers[0].Ports = []corev1.ContainerPort{}
		serviceDefinition.Spec.Ports = []corev1.ServicePort{}

		for _, p := range def.Ports {
			podDefinition.Spec.Containers[0].Ports = append(podDefinition.Spec.Containers[0].Ports,
				corev1.ContainerPort{
					ContainerPort: int32(p),
				},
			)

			svcPort := corev1.ServicePort{
				Port:       int32(p),
				TargetPort: intstr.FromInt(p),
			}

			if def.AppProtocol != "" {
				if ver, err := td.getKubernetesServerVersionNumber(); err != nil {
					svcPort.Name = fmt.Sprintf("%s-%d", def.AppProtocol, p) // use named port with AppProtocol
				} else {
					// use appProtocol field in servicePort if k8s server version >= 1.19
					if ver[0] >= 1 && ver[1] >= 19 {
						svcPort.AppProtocol = &def.AppProtocol // set the appProtocol field
					} else {
						svcPort.Name = fmt.Sprintf("%s-%d", def.AppProtocol, p) // use named port with AppProtocol
					}
				}
			}

			serviceDefinition.Spec.Ports = append(serviceDefinition.Spec.Ports, svcPort)
		}
	}

	return serviceAccountDefinition, podDefinition, serviceDefinition
}

// getKubernetesServerVersionNumber returns the version number in chunks, ex. v1.19.3 => [1, 19, 3]
func (td *OsmTestData) getKubernetesServerVersionNumber() ([]int, error) {
	version, err := td.Client.Discovery().ServerVersion()
	if err != nil {
		return nil, errors.Errorf("Error getting K8s server version: %s", err)
	}

	ver, err := goversion.NewVersion(version.String())
	if err != nil {
		return nil, errors.Errorf("Error parsing k8s server version %s: %s", version.String(), err)
	}

	return ver.Segments(), nil
}

// SimpleDeploymentAppDef defines some parametrization to create a deployment-based application from template
type SimpleDeploymentAppDef struct {
	Namespace    string
	Name         string
	Image        string
	ReplicaCount int32
	Command      []string
	Args         []string
	Ports        []int
}

// SimpleDeploymentApp creates returns a set of k8s typed definitions for a deployment-based k8s definition.
// Includes Deployment, Service and ServiceAccount types
func (td *OsmTestData) SimpleDeploymentApp(def SimpleDeploymentAppDef) (corev1.ServiceAccount, appsv1.Deployment, corev1.Service) {
	serviceAccountDefinition := corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      def.Name,
			Namespace: def.Namespace,
		},
	}

	// Required, as replica count is a pointer to distinguish between 0 and not specified
	replicaCountExplicitDeclaration := def.ReplicaCount

	deploymentDefinition := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      def.Name,
			Namespace: def.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicaCountExplicitDeclaration,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": def.Name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": def.Name,
					},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: def.Name,
					Containers: []corev1.Container{
						{
							Name:            def.Name,
							Image:           def.Image,
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

	if def.Command != nil && len(def.Command) > 0 {
		deploymentDefinition.Spec.Template.Spec.Containers[0].Command = def.Command
	}
	if def.Args != nil && len(def.Args) > 0 {
		deploymentDefinition.Spec.Template.Spec.Containers[0].Args = def.Args
	}

	serviceDefinition := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      def.Name,
			Namespace: def.Namespace,
			Labels: map[string]string{
				"app": def.Name,
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": def.Name,
			},
		},
	}

	if def.Ports != nil && len(def.Ports) > 0 {
		deploymentDefinition.Spec.Template.Spec.Containers[0].Ports = []corev1.ContainerPort{}
		serviceDefinition.Spec.Ports = []corev1.ServicePort{}

		for _, p := range def.Ports {
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

// GetGrafanaPodHandle generic func to forward a grafana pod and returns a handler pointing to the locally forwarded resource
func (td *OsmTestData) GetGrafanaPodHandle(ns string, grafanaPodName string, port uint16) (*Grafana, error) {
	portForwarder, err := k8s.NewPortForwarder(td.RestConfig, td.Client, grafanaPodName, ns, port, port)
	if err != nil {
		return nil, errors.Errorf("Error setting up port forwarding: %s", err)
	}

	err = portForwarder.Start(func(pf *k8s.PortForwarder) error {
		return nil
	})
	if err != nil {
		return nil, errors.Errorf("Could not start forwarding: %s", err)
	}

	return &Grafana{
		Schema:   "http",
		Hostname: "localhost",
		Port:     port,
		User:     "admin", // default value of grafana deployment
		Password: "admin", // default value of grafana deployment
		pfwd:     portForwarder,
	}, nil
}

// GetPrometheusPodHandle generic func to forward a prometheus pod and returns a handler pointing to the locally forwarded resource
func (td *OsmTestData) GetPrometheusPodHandle(ns string, prometheusPodName string, port uint16) (*Prometheus, error) {
	portForwarder, err := k8s.NewPortForwarder(td.RestConfig, td.Client, prometheusPodName, ns, port, port)
	if err != nil {
		return nil, errors.Errorf("Error setting up port forwarding: %s", err)
	}

	err = portForwarder.Start(func(pf *k8s.PortForwarder) error {
		return nil
	})
	if err != nil {
		return nil, errors.Errorf("Could not start forwarding: %s", err)
	}

	client, err := api.NewClient(api.Config{
		Address: fmt.Sprintf("http://localhost:%d", port),
	})
	if err != nil {
		return nil, err
	}

	v1api := v1.NewAPI(client)

	return &Prometheus{
		Client: client,
		API:    v1api,
		pfwd:   portForwarder,
	}, nil
}

// GetOSMPrometheusHandle convenience wrapper, will get the Prometheus instance regularly deployed
// by OSM installation in test <OsmNamespace>
func (td *OsmTestData) GetOSMPrometheusHandle() (*Prometheus, error) {
	prometheusPod, err := Td.GetPodsForLabel(Td.OsmNamespace, metav1.LabelSelector{
		MatchLabels: map[string]string{
			"app": OsmPrometheusAppLabel,
		},
	})
	if err != nil || len(prometheusPod) == 0 {
		return nil, errors.Errorf("Error getting Prometheus pods: %v (prom pods len: %d)", err, len(prometheusPod))
	}
	pHandle, err := Td.GetPrometheusPodHandle(prometheusPod[0].Namespace, prometheusPod[0].Name, DefaultOsmPrometheusPort)
	if err != nil {
		return nil, err
	}

	return pHandle, nil
}

// GetOSMGrafanaHandle convenience wrapper, will get the Grafana instance regularly deployed
// by OSM installation in test <OsmNamespace>
func (td *OsmTestData) GetOSMGrafanaHandle() (*Grafana, error) {
	grafanaPod, err := Td.GetPodsForLabel(Td.OsmNamespace, metav1.LabelSelector{
		MatchLabels: map[string]string{
			"app": OsmGrafanaAppLabel,
		},
	})
	if err != nil || len(grafanaPod) == 0 {
		return nil, errors.Errorf("Error getting Grafana pods: %v (graf pods len: %d)", err, len(grafanaPod))
	}
	gHandle, err := Td.GetGrafanaPodHandle(grafanaPod[0].Namespace, grafanaPod[0].Name, DefaultOsmGrafanaPort)
	if err != nil {
		return nil, err
	}
	return gHandle, nil
}
