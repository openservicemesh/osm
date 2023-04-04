package plugin

import (
	"context"
	"os"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	kubescheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	// KubeScheme returns a scheme will all known related types added
	KubeScheme = kubeScheme()

	podRetrievalMaxRetries = 30
	podRetrievalInterval   = 1 * time.Second
)

func kubeScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	utilruntime.Must(kubescheme.AddToScheme(scheme))
	return scheme
}

// newKubeClient returns a Kubernetes client
func newKubeClient(conf Config) (*kubernetes.Clientset, error) {
	// Some config can be passed in a kubeconfig file
	kubeconfig := conf.Kubernetes.Kubeconfig

	config, err := DefaultRestConfig(kubeconfig, "")
	if err != nil {
		log.Error().Msgf("Failed setting up kubernetes client with kubeconfig %s", kubeconfig)
		return nil, err
	}

	log.Debug().Msgf("osm-cni set up kubernetes client with kubeconfig %s", kubeconfig)

	// Create the clientset
	return kubernetes.NewForConfig(config)
}

// DefaultRestConfig returns the rest.Config for the given kube config file and context.
func DefaultRestConfig(kubeconfig, configContext string, fns ...func(*rest.Config)) (*rest.Config, error) {
	config, err := BuildClientConfig(kubeconfig, configContext)
	if err != nil {
		return nil, err
	}
	config = SetRestDefaults(config)

	for _, fn := range fns {
		fn(config)
	}

	return config, nil
}

// BuildClientConfig builds a client rest config from a kubeconfig filepath and context.
// It overrides the current context with the one provided (empty to use default).
//
// This is a modified version of k8s.io/client-go/tools/clientcmd/BuildConfigFromFlags with the
// difference that it loads default configs if not running in-cluster.
func BuildClientConfig(kubeconfig, context string) (*rest.Config, error) {
	c, err := BuildClientCmd(kubeconfig, context).ClientConfig()
	if err != nil {
		return nil, err
	}
	return SetRestDefaults(c), nil
}

// BuildClientCmd builds a client cmd config from a kubeconfig filepath and context.
// It overrides the current context with the one provided (empty to use default).
//
// This is a modified version of k8s.io/client-go/tools/clientcmd/BuildConfigFromFlags with the
// difference that it loads default configs if not running in-cluster.
func BuildClientCmd(kubeconfig, context string, overrides ...func(*clientcmd.ConfigOverrides)) clientcmd.ClientConfig {
	if kubeconfig != "" {
		info, err := os.Stat(kubeconfig)
		if err != nil || info.Size() == 0 {
			// If the specified kubeconfig doesn't exists / empty file / any other error
			// from file stat, fall back to default
			kubeconfig = ""
		}
	}

	// Config loading rules:
	// 1. kubeconfig if it not empty string
	// 2. Config(s) in KUBECONFIG environment variable
	// 3. In cluster config if running in-cluster
	// 4. Use $HOME/.kube/config
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	loadingRules.DefaultClientConfig = &clientcmd.DefaultClientConfig
	loadingRules.ExplicitPath = kubeconfig
	configOverrides := &clientcmd.ConfigOverrides{
		ClusterDefaults: clientcmd.ClusterDefaults,
		CurrentContext:  context,
	}

	for _, fn := range overrides {
		fn(configOverrides)
	}

	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)
}

// SetRestDefaults is a helper function that sets default values for the given rest.Config.
// This function is idempotent.
func SetRestDefaults(config *rest.Config) *rest.Config {
	if config.GroupVersion == nil || config.GroupVersion.Empty() {
		config.GroupVersion = &corev1.SchemeGroupVersion
	}
	if len(config.APIPath) == 0 {
		if len(config.GroupVersion.Group) == 0 {
			config.APIPath = "/api"
		} else {
			config.APIPath = "/apis"
		}
	}
	if len(config.ContentType) == 0 {
		config.ContentType = runtime.ContentTypeJSON
	}
	if config.NegotiatedSerializer == nil {
		// This codec factory ensures the resources are not converted. Therefore, resources
		// will not be round-tripped through internal versions. Defaulting does not happen
		// on the client.
		config.NegotiatedSerializer = serializer.NewCodecFactory(KubeScheme).WithoutConversion()
	}

	return config
}

// getKubePodInfo returns information of a POD
func getKubePodInfo(client *kubernetes.Clientset, podName, podNamespace string) (*podInfo, error) {
	pod, err := client.CoreV1().Pods(podNamespace).Get(context.TODO(), podName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	proxyContainerName := "sidecar"

	pi := &podInfo{
		Containers:        make([]string, len(pod.Spec.Containers)),
		Annotations:       pod.Annotations,
		ProxyEnvironments: make(map[string]string),
	}
	for containerIdx, container := range pod.Spec.Containers {
		log.Debug().Msgf("Inspecting pod %v/%v container %v", podNamespace, podName, container.Name)
		pi.Containers[containerIdx] = container.Name

		if container.Name == proxyContainerName {
			for _, e := range container.Env {
				pi.ProxyEnvironments[e.Name] = e.Value
			}
			continue
		}
	}
	log.Debug().Msgf("Pod %v/%v info: \n%+v", podNamespace, podName, pi)

	return pi, nil
}
