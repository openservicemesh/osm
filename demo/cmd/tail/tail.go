package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Azure/go-autorest/autorest/to"
	"github.com/rs/zerolog/log"
	"k8s.io/api/admissionregistration/v1beta1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/open-service-mesh/osm/demo/cmd/common"
)

var (
	waitForPod = 5 * time.Second
)

var errNoPodsFound = errors.New("no pods found")

const (
	// KubeConfigEnvVar is the environment variable for KUBECONFIG.
	KubeConfigEnvVar = "KUBECONFIG"

	// OSMNamespaceEnvVar is the environment variable for the OSM namespace.
	OSMNamespaceEnvVar = "K8S_NAMESPACE"

	// BookbuyerNamespaceEnvVar is the environment variable for the Bookbuyer namespace.
	BookbuyerNamespaceEnvVar = "BOOKBUYER_NAMESPACE"

	// BookthiefNamespaceEnvVar is the environment variable for the Bookbuyer namespace.
	BookthiefNamespaceEnvVar = "BOOKTHIEF_NAMESPACE"

	// BookstoreNamespaceEnvVar is the environment variable for the Bookbuyer namespace.
	BookstoreNamespaceEnvVar = "BOOKSTORE_NAMESPACE"

	// WaitForPodTimeSecondsEnvVar is the environment variable for the time we will wait on the pod to be ready.
	WaitForPodTimeSecondsEnvVar = "WAIT_FOR_POD_TIME_SECONDS"
)

func main() {
	osmNS := os.Getenv(OSMNamespaceEnvVar)
	bookbuyerNS := os.Getenv(BookbuyerNamespaceEnvVar)
	bookthiefNS := os.Getenv(BookthiefNamespaceEnvVar)
	bookstoreNS := os.Getenv(BookstoreNamespaceEnvVar)
	totalWaitString := os.Getenv(WaitForPodTimeSecondsEnvVar)
	totalWait, err := strconv.ParseInt(totalWaitString, 10, 32)
	if err != nil {
		log.Fatal().Err(err).Msgf("Could not convert environment variable %s='%s' to int", WaitForPodTimeSecondsEnvVar, totalWaitString)
	}
	totalWaitSeconds := time.Duration(totalWait) * time.Second
	bookBuyerContainerName := "bookbuyer"
	bookBuyerSelector := "app=bookbuyer"
	bookThiefContainerName := "bookthief"
	bookThiefSelector := "app=bookthief"
	adsPodSelector := "app=ads"

	namespaces := []string{
		bookbuyerNS,
		bookthiefNS,
		bookstoreNS,
		osmNS,
	}

	fmt.Printf("Tail looking for containers:namespace - %s:%s and %s:%s\n", bookBuyerContainerName, bookbuyerNS, bookThiefContainerName, bookthiefNS)
	if bookbuyerNS == "" || bookthiefNS == "" {
		fmt.Printf("Namespace cannot be empty, bookbuyer=%s, bookthief=%s\n", bookbuyerNS, bookthiefNS)
		os.Exit(1)
	}
	clientset := getClient()
	bookbuyerReady := make(chan struct{})
	bookthiefReady := make(chan struct{})
	startedWaiting := time.Now()

	go func() {
	Run:
		for {
			if time.Since(startedWaiting) >= totalWaitSeconds {
				fmt.Printf("Waited for bookbuyer pod to become ready for %+v; Didn't happen", totalWait)
				os.Exit(1)
			}
			bookBuyerPodName, err := getPodName(bookbuyerNS, bookBuyerSelector)
			if err != nil {
				fmt.Println("Error getting bookbuyer pod: ", err)
				time.Sleep(waitForPod)
				// Pod might not be up yet, try again
				continue
			}
			bookBuyerPod, err := clientset.CoreV1().Pods(bookbuyerNS).Get(bookBuyerPodName, metav1.GetOptions{})
			if err != nil {
				fmt.Printf("Error getting pod %s/%s: %s\n", bookbuyerNS, bookBuyerPodName, err)
				os.Exit(1)
			}
			for _, container := range bookBuyerPod.Status.ContainerStatuses {
				if container.State.Waiting != nil && container.State.Waiting.Reason == "PodInitializing" {
					fmt.Printf("Pod %s/%s is still initializing; Waiting %+v (%+v/%+v)\n", bookbuyerNS, bookBuyerPodName, waitForPod, time.Since(startedWaiting), totalWait)
					time.Sleep(waitForPod)
				} else {
					fmt.Println("Bookbuyer pod init done")
					close(bookbuyerReady)
					break Run
				}
			}
		}
	}()

	go func() {
	Run:
		for {
			if time.Since(startedWaiting) >= totalWaitSeconds {
				fmt.Printf("Waited for bookthief pod to become ready for %+v; Didn't happen", totalWait)
				os.Exit(1)
			}
			bookThiefPodName, err := getPodName(bookthiefNS, bookThiefSelector)
			if err != nil {
				fmt.Println("Error getting Bookthief pod: ", err)
				time.Sleep(waitForPod)
				// Pod might not be up yet, try again
				continue
			}
			bookThiefPod, err := clientset.CoreV1().Pods(bookthiefNS).Get(bookThiefPodName, metav1.GetOptions{})
			if err != nil {
				fmt.Printf("Error getting pod %s/%s: %s\n", bookthiefNS, bookThiefPodName, err)
				os.Exit(1)
			}
			for _, container := range bookThiefPod.Status.ContainerStatuses {
				if container.State.Waiting != nil && container.State.Waiting.Reason == "PodInitializing" {
					fmt.Printf("Pod %s/%s is still initializing; Waiting %+v (%+v/%+v)\n", bookthiefNS, bookThiefPodName, waitForPod, time.Since(startedWaiting), totalWait)
					time.Sleep(waitForPod)
				} else {
					fmt.Println("Bookthief pod init done")
					close(bookthiefReady)
					break Run
				}
			}
		}
	}()

	<-bookbuyerReady
	<-bookthiefReady

	bookBuyerPodName, err := getPodName(bookbuyerNS, bookBuyerSelector)
	if err != nil {
		fmt.Println("Error getting bookbuyer pod after pod being ready: ", err)
		os.Exit(1)
	}
	bookThiefPodName, err := getPodName(bookthiefNS, bookThiefSelector)
	if err != nil {
		fmt.Println("Error getting bookthief pod after pod being ready: ", err)
		os.Exit(1)
	}

	// Poll for success
	for {
		if time.Since(startedWaiting) >= totalWaitSeconds {
			// failure
			break
		}
		bookBuyerLogs := getPodLogs(bookbuyerNS, bookBuyerPodName, bookBuyerContainerName, false)
		bookThiefLogs := getPodLogs(bookthiefNS, bookThiefPodName, bookThiefContainerName, false)

		if strings.Contains(bookBuyerLogs, common.Success) && strings.Contains(bookThiefLogs, common.Success) {
			fmt.Println("The test succeeded")
			deleteNamespaces(clientset, namespaces...)
			deleteWebhooks(clientset, namespaces...)
			os.Exit(0)
		}
	}

	fmt.Println("The test failed")

	bookBuyerLogs := getPodLogs(bookbuyerNS, bookBuyerPodName, bookBuyerContainerName, false)
	bookThiefLogs := getPodLogs(bookthiefNS, bookThiefPodName, bookThiefContainerName, false)
	fmt.Println("-------- Bookbuyer LOGS --------\n", bookBuyerLogs)
	fmt.Println("-------- Bookthief LOGS --------\n", bookThiefLogs)

	adsPodName, err := getPodName(osmNS, adsPodSelector)
	if err != nil {
		log.Fatal().Err(err).Msgf("Error getting ADS pods with selector %s in namespace %s", adsPodName, osmNS)
	}
	fmt.Println("-------- ADS LOGS --------\n", getPodLogs(osmNS, adsPodName, "", false))
	os.Exit(1)
}

func deleteNamespaces(client *kubernetes.Clientset, namespaces ...string) {
	deleteOptions := &metav1.DeleteOptions{
		GracePeriodSeconds: to.Int64Ptr(0),
	}

	for _, ns := range namespaces {
		if err := client.CoreV1().Namespaces().Delete(ns, deleteOptions); err != nil {
			log.Error().Err(err).Msgf("Error deleting namespace %s", ns)
		}
		log.Info().Msgf("Deleted namespace: %s", ns)
	}
}

func deleteWebhooks(client *kubernetes.Clientset, namespaces ...string) {
	deleteOptions := &metav1.DeleteOptions{
		GracePeriodSeconds: to.Int64Ptr(0),
	}

	var webhooks *v1beta1.MutatingWebhookConfigurationList
	var err error
	webhooks, err = client.AdmissionregistrationV1beta1().MutatingWebhookConfigurations().List(metav1.ListOptions{})
	if err != nil {
		log.Error().Err(err).Msg("Error listing webhooks")
	}

	for _, webhook := range webhooks.Items {
		for _, ns := range namespaces {
			// Convention is - the webhook name is prefixed with the namespace where OSM is.
			if !strings.HasPrefix(webhook.Name, ns) {
				continue
			}
			if err := client.AdmissionregistrationV1beta1().MutatingWebhookConfigurations().Delete(webhook.Name, deleteOptions); err != nil {
				log.Error().Err(err).Msgf("Error deleting webhook %s", webhook.Name)
			}
			log.Info().Msgf("Deleted mutating webhook: %s", webhook.Name)
		}
	}
}

func getPodName(namespace, selector string) (string, error) {
	clientset := getClient()

	podList, err := clientset.CoreV1().Pods(namespace).List(metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		return "", err
	}

	if len(podList.Items) == 0 {
		log.Error().Msgf("Zero pods found for selector %s in namespace %s", selector, namespace)
		return "", errNoPodsFound
	}

	sort.SliceStable(podList.Items, func(i, j int) bool {
		p1 := podList.Items[i].CreationTimestamp.UnixNano()
		p2 := podList.Items[j].CreationTimestamp.UnixNano()
		return p1 > p2
	})

	return podList.Items[0].Name, nil
}

func getPodLogs(namespace string, podName string, containerName string, follow bool) string {
	clientset := getClient()
	sinceTime := metav1.NewTime(time.Now().Add(-2 * time.Second))
	options := &v1.PodLogOptions{
		Container: containerName,
		Follow:    follow,
		SinceTime: &sinceTime,
	}

	rc, err := clientset.CoreV1().Pods(namespace).GetLogs(podName, options).Stream()
	if err != nil {
		fmt.Println("Error in opening stream: ", err)
		os.Exit(1)
	}

	defer rc.Close()
	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(rc)
	if err != nil {
		log.Error().Err(err).Msg("Error reading from pod logs stream")
	}
	return buf.String()
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
