package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/glog"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	// KubeConfigEnvVar is the environment variable name for KUBECONFIG
	KubeConfigEnvVar = "KUBECONFIG"

	// KubeNamespaceEnvVar is the environment variable name for the K8s namespace
	KubeNamespaceEnvVar = "K8S_NAMESPACE"
)

var (
	staleIfOlderThan = 12 * time.Hour
)

func main() {
	clientset := getClient()

	namespaces, err := clientset.CoreV1().Namespaces().List(metav1.ListOptions{})
	if err != nil {
		fmt.Println("Error listing namespaces: ", err)
		os.Exit(1)
	}

	var namespacesToDelete []v1.Namespace
	for _, ns := range namespaces.Items {
		isStale := time.Now().Sub(ns.CreationTimestamp.UTC()) > staleIfOlderThan
		if isStale && strings.HasPrefix(ns.Name, "ci-") {
			namespacesToDelete = append(namespacesToDelete, ns)
		}
	}

	if len(namespacesToDelete) == 0 {
		fmt.Println("No stale namespaces to cleanup.")
		return
	}

	for _, ns := range namespacesToDelete {
		if err = clientset.CoreV1().Namespaces().Delete(ns.Name, &metav1.DeleteOptions{GracePeriodSeconds: to.Int64Ptr(0)}); err != nil {
			glog.Errorf("Error deleting namespace %s: %s", ns.Name, err)
		}
		glog.Infof("Deleted namespace: %s", ns.Name)
	}
}

func getClient() *kubernetes.Clientset {
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
