package main

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/Azure/go-autorest/autorest/to"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/open-service-mesh/osm/pkg/logger"
)

var log = logger.New("ci/maestro")

const (
	// KubeConfigEnvVar is the environment variable name for KUBECONFIG
	KubeConfigEnvVar = "KUBECONFIG"
)

var (
	staleIfOlderThan = 15 * time.Minute
)

func main() {
	clientset := getClient()

	webHooks, err := clientset.AdmissionregistrationV1beta1().MutatingWebhookConfigurations().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		log.Error().Err(err).Msg("Error listing mutating webhooks")
	}
	namespaces, err := clientset.CoreV1().Namespaces().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		log.Error().Err(err).Msgf("Error listing namespaces")
		os.Exit(1)
	}

	deleteOptions := metav1.DeleteOptions{
		GracePeriodSeconds: to.Int64Ptr(0),
	}

	// Delete stale namespaces
	for _, ns := range namespaces.Items {
		isStale := time.Since(ns.CreationTimestamp.UTC()) > staleIfOlderThan
		if isStale && strings.HasPrefix(ns.Name, "ci-") {
			log.Info().Msgf("Deleting namespace: %s", ns.Name)
			if err = clientset.CoreV1().Namespaces().Delete(context.Background(), ns.Name, deleteOptions); err != nil {
				log.Error().Err(err).Msgf("Error deleting namespace %s", ns.Name)
			}
		} else {
			log.Info().Msgf("Keep namespace %s - it is not older than %+v", ns.Name, staleIfOlderThan)
		}
	}

	// Delete stale mutating webhook configurations
	for _, webhook := range webHooks.Items {
		isStale := time.Since(webhook.CreationTimestamp.UTC()) > staleIfOlderThan
		if isStale && strings.HasPrefix(webhook.Name, "ci-") {
			log.Info().Msgf("Deleting mutating webhook: %s", webhook.Name)
			if err = clientset.AdmissionregistrationV1beta1().MutatingWebhookConfigurations().Delete(context.Background(), webhook.Name, deleteOptions); err != nil {
				log.Error().Err(err).Msgf("Error deleting webhook %s", webhook.Name)
			}
		} else {
			log.Info().Msgf("Keep webhook %s - it is not older than %+v", webhook.Name, staleIfOlderThan)
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
