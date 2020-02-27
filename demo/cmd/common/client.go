package common

import (
	"fmt"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"os"
)

// GetClient returns a Kubernetes clientset.
func GetClient() *kubernetes.Clientset {
	var kubeConfig *rest.Config
	var err error
	kubeConfigFile := os.Getenv(KubeConfigEnvVar)
	if kubeConfigFile != "" {
		kubeConfig, err = clientcmd.BuildConfigFromFlags("", kubeConfigFile)
		if err != nil {
			fmt.Printf("Error fetching Kubernetes config. Ensure correctness of CLI argument 'kubeconfig=%s': %s", kubeConfigFile, err)
			os.Exit(1)
		}
	} else {
		// creates the in-cluster config
		kubeConfig, err = rest.InClusterConfig()
		if err != nil {
			fmt.Printf("Error generating Kubernetes config: %s", err)
			os.Exit(1)
		}
	}

	clientset, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		fmt.Println("error in getting access to K8S")
		os.Exit(1)
	}
	return clientset
}
