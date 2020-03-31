package main

import (
	"os"
	"strings"
	"time"

	"github.com/Azure/go-autorest/autorest/to"
	"github.com/rs/zerolog/log"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	// KubeConfigEnvVar is the environment variable name for KUBECONFIG
	KubeConfigEnvVar = "KUBECONFIG"
)

var (
	staleIfOlderThan = 24 * time.Hour
)

func main() {
	clientset := getClient()

	webHooks, err := clientset.AdmissionregistrationV1beta1().MutatingWebhookConfigurations().List(metav1.ListOptions{})
	if err != nil {
		log.Error().Err(err).Msg("Error listing mutating webhooks")
	}
	namespaces, err := clientset.CoreV1().Namespaces().List(metav1.ListOptions{})
	if err != nil {
		log.Error().Err(err).Msgf("Error listing namespaces")
		os.Exit(1)
	}

	var namespacesToDelete []v1.Namespace
	for _, ns := range namespaces.Items {
		isStale := time.Since(ns.CreationTimestamp.UTC()) > staleIfOlderThan
		if isStale && strings.HasPrefix(ns.Name, "ci-") {
			namespacesToDelete = append(namespacesToDelete, ns)
		}
	}

	if len(namespacesToDelete) == 0 {
		log.Info().Msg("No stale namespaces to cleanup.")
		return
	}

	deleteOptions := &metav1.DeleteOptions{
		GracePeriodSeconds: to.Int64Ptr(0),
	}

	for _, ns := range namespacesToDelete {
		if err = clientset.CoreV1().Namespaces().Delete(ns.Name, deleteOptions); err != nil {
			log.Error().Err(err).Msgf("Error deleting namespace %s", ns.Name)
		}
		log.Info().Msgf("Deleted namespace: %s", ns.Name)
		for _, webhook := range webHooks.Items {
			// Convention is - the webhook name is prefixed with the namespace where OSM is.
			if !strings.HasPrefix(webhook.Name, ns.Name) {
				continue
			}
			opts := metav1.DeleteOptions{}
			if err = clientset.AdmissionregistrationV1beta1().MutatingWebhookConfigurations().Delete(webhook.Name, &opts); err != nil {
				log.Error().Err(err).Msgf("Error deleting webhook %s", webhook.Name)
			}
			log.Info().Msgf("Deleted mutating webhook: %s", webhook.Name)
		}
	}
}

func getClient() *kubernetes.Clientset {
	var kubeConfig *rest.Config
	var err error
	kubeConfigFile := os.Getenv(KubeConfigEnvVar)
	if kubeConfigFile != "" {
		kubeConfig, err = clientcmd.BuildConfigFromFlags("", kubeConfigFile)
		if err != nil {
			log.Error().Err(err).Msgf("Error fetching Kubernetes config. Ensure correctness of CLI argument 'kubeconfig=%s", kubeConfigFile)
		}
	} else {
		// creates the in-cluster config
		kubeConfig, err = rest.InClusterConfig()
		if err != nil {
			log.Error().Err(err).Msg("Error generating Kubernetes config")
		}
	}

	clientset, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		log.Error().Msgf("error in getting access to K8S")
	}
	return clientset
}
