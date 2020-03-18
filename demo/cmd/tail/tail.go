package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/open-service-mesh/osm/demo/cmd/common"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	waitForPod = 5 * time.Second
)

const (
	// KubeConfigEnvVar is the environment variable for KUBECONFIG.
	KubeConfigEnvVar = "KUBECONFIG"

	// OSMNamespaceEnvVar is the environment variable for the OSM namespace.
	OSMNamespaceEnvVar = "K8S_NAMESPACE"

	// BookbuyerNamespaceEnvVar is the environment variable for the Bookbuyer namespace.
	BookbuyerNamespaceEnvVar = "BOOKBUYER_NAMESPACE"

	// BookthiefNamespaceEnvVar is the environment variable for the Bookbuyer namespace.
	BookthiefNamespaceEnvVar = "BOOKTHIEF_NAMESPACE"

	// WaitForPodTimeSecondsEnvVar is the environment variable for the time we will wait on the pod to be ready.
	WaitForPodTimeSecondsEnvVar = "WAIT_FOR_POD_TIME_SECONDS"
)

func main() {
	osmNS := os.Getenv(OSMNamespaceEnvVar)
	bookbuyerNS := os.Getenv(BookbuyerNamespaceEnvVar)
	bookthiefNS := os.Getenv(BookthiefNamespaceEnvVar)
	totalWaitString := os.Getenv(WaitForPodTimeSecondsEnvVar)
	totalWait, err := strconv.ParseInt(totalWaitString, 10, 32)
	if err != nil {
		glog.Fatalf("Could not convert environment variable %s='%s' to int: %+v", WaitForPodTimeSecondsEnvVar, totalWaitString, err)
	}
	totalWaitSeconds := time.Duration(totalWait) * time.Second
	bookBuyerContainerName := "bookbuyer"
	bookBuyerSelector := "app=bookbuyer"
	bookThiefContainerName := "bookthief"
	bookThiefSelector := "app=bookthief"
	adsPodSelector := "app=ads"

	fmt.Printf("Tail looking for containers:namespace - %s:%s and %s:%s\n", bookBuyerContainerName, bookbuyerNS, bookThiefContainerName, bookthiefNS)
	if bookbuyerNS == "" || bookthiefNS == "" {
		fmt.Printf("Namespace cannot be empty, bookbuyer=%s, bookthief=%s\n", bookbuyerNS, bookthiefNS)
		os.Exit(1)
	}
	clientset := getClient()

	bookBuyerPodName, err := getPodName(bookbuyerNS, bookBuyerSelector)
	if err != nil {
		glog.Fatal("Error getting Bookbuyer pod: ", err)
	}

	bookThiefPodName, err := getPodName(bookthiefNS, bookThiefSelector)
	if err != nil {
		glog.Fatal("Error getting Bookthief pod: ", err)
	}
	startedWaiting := time.Now()
	go func() {
	Run:
		for {
			bookBuyerPod, err := clientset.CoreV1().Pods(bookbuyerNS).Get(bookBuyerPodName, metav1.GetOptions{})
			if err != nil {
				fmt.Printf("Error getting pod %s/%s: %s\n", bookbuyerNS, bookBuyerPodName, err)
				os.Exit(1)
			}
			for _, container := range bookBuyerPod.Status.ContainerStatuses {
				if container.State.Waiting != nil && container.State.Waiting.Reason == "PodInitializing" {
					if time.Now().Sub(startedWaiting) >= totalWaitSeconds {
						fmt.Printf("Waited for pod %s to become ready for %+v; Didn't happen", bookBuyerPodName, totalWait)
						os.Exit(1)
					}
					fmt.Printf("Pod %s/%s is still initializing; Waiting %+v (%+v/%+v)\n", bookbuyerNS, bookBuyerPodName, waitForPod, time.Now().Sub(startedWaiting), totalWait)
					time.Sleep(waitForPod)
				} else {
					break Run
				}
			}
		}
	}()

	go func() {
	Run:
		for {
			bookThiefPod, err := clientset.CoreV1().Pods(bookthiefNS).Get(bookThiefPodName, metav1.GetOptions{})
			if err != nil {
				fmt.Printf("Error getting pod %s/%s: %s\n", bookthiefNS, bookBuyerPodName, err)
				os.Exit(1)
			}
			for _, container := range bookThiefPod.Status.ContainerStatuses {
				if container.State.Waiting != nil && container.State.Waiting.Reason == "PodInitializing" {
					if time.Now().Sub(startedWaiting) >= totalWaitSeconds {
						fmt.Printf("Waited for pod %s to become ready for %+v; Didn't happen", bookThiefPodName, totalWait)
						os.Exit(1)
					}
					fmt.Printf("Pod %s/%s is still initializing; Waiting %+v (%+v/%+v)\n", bookthiefNS, bookThiefPodName, waitForPod, time.Now().Sub(startedWaiting), totalWait)
					time.Sleep(waitForPod)
				} else {
					break Run
				}
			}
		}
	}()

	bookBuyerLogs := getPodLogs(bookbuyerNS, bookBuyerPodName, bookBuyerContainerName, true)
	bookThiefLogs := getPodLogs(bookthiefNS, bookThiefPodName, bookThiefContainerName, true)
	if strings.HasSuffix(bookBuyerLogs, common.Success) && strings.HasSuffix(bookThiefLogs, common.Success) {
		fmt.Println("The test succeeded")
		os.Exit(0)
	}
	fmt.Println(bookBuyerLogs)
	fmt.Println(bookThiefLogs)

	adsPodName, err := getPodName(osmNS, adsPodSelector)
	if err != nil {
		glog.Fatalf("Error getting ADS pods with selector %s in namespace %s: %s", adsPodName, osmNS, err)
	}
	fmt.Println("-------- ADS LOGS --------\n", getPodLogs(osmNS, adsPodName, "", false))
	os.Exit(1)
}

func getPodName(namespace, selector string) (string, error) {
	clientset := getClient()

	podList, err := clientset.CoreV1().Pods(namespace).List(metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		return "", err
	}

	if len(podList.Items) == 0 {
		return "", errors.New("Zero pods found")
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
	options := &v1.PodLogOptions{
		Container: containerName,
		Follow:    follow,
	}
	req := clientset.CoreV1().Pods(namespace).GetLogs(podName, options)
	podLogs, err := req.Stream()
	if err != nil {
		fmt.Println("Error in opening stream: ", err)
		os.Exit(1)
	}
	defer podLogs.Close()

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, podLogs)
	if err != nil {
		fmt.Println("error in copy information from podLogs to buf")
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
