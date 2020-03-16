package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/open-service-mesh/osm/demo/cmd/common"
	"k8s.io/api/core/v1"
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

	// KubeNamespaceEnvVar is the environment variable for the Kubernetes namespace.
	KubeNamespaceEnvVar = "K8S_NAMESPACE"

	// WaitForPodTimeSecondsEnvVar is the environment variable for the time we will wait on the pod to be ready.
	WaitForPodTimeSecondsEnvVar = "WAIT_FOR_POD_TIME_SECONDS"
)

func main() {
	namespace := os.Getenv(KubeNamespaceEnvVar)
	totalWaitString := os.Getenv(WaitForPodTimeSecondsEnvVar)
	totalWait, err := strconv.ParseInt(totalWaitString, 10, 32)
	if err != nil {
		glog.Fatalf("Could not convert environment variable %s='%s' to int: %+v", WaitForPodTimeSecondsEnvVar, totalWaitString, err)
	}
	totalWaitSeconds := time.Duration(totalWait) * time.Second
	containerName := "bookbuyer"
	labelSelector := "app=bookbuyer"

	fmt.Printf("Tail looking for container %s in namespace %s\n", containerName, namespace)
	if namespace == "" {
		fmt.Println("Empty namespace")
		os.Exit(1)
	}
	clientset := getClient()

	podList, err := clientset.CoreV1().Pods(namespace).List(metav1.ListOptions{LabelSelector: labelSelector})
	if err != nil {
		fmt.Println("Error fetching pods: ", err)
		os.Exit(1)
	}

	if len(podList.Items) == 0 {
		fmt.Println("Zero pods found")
		os.Exit(1)
	}

	sort.SliceStable(podList.Items, func(i, j int) bool {
		p1 := podList.Items[i].CreationTimestamp.UnixNano()
		p2 := podList.Items[j].CreationTimestamp.UnixNano()
		return p1 > p2
	})

	podName := podList.Items[0].Name
	startedWaiting := time.Now()
Run:
	for {
		pod, err := clientset.CoreV1().Pods(namespace).Get(podName, metav1.GetOptions{})
		if err != nil {
			fmt.Printf("Error getting pod %s/%s: %s\n", namespace, podName, err)
			os.Exit(1)
		}
		for _, container := range pod.Status.ContainerStatuses {
			if container.State.Waiting != nil && container.State.Waiting.Reason == "PodInitializing" {
				if time.Now().Sub(startedWaiting) >= totalWaitSeconds {
					fmt.Printf("Waited for pod %s to become ready for %+v; Didn't happen", podName, totalWait)
					os.Exit(1)
				}
				fmt.Printf("Pod %s/%s is still initializing; Waiting %+v (%+v/%+v)\n", namespace, podName, waitForPod, time.Now().Sub(startedWaiting), totalWait)
				time.Sleep(waitForPod)
			} else {
				break Run
			}
		}
	}
	options := &v1.PodLogOptions{
		Container: containerName,
		Follow:    true,
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
	logs := buf.String()
	if strings.HasSuffix(logs, common.Success) {
		fmt.Println("The test succeeded")
		os.Exit(0)
	}
	fmt.Println(logs)
	os.Exit(1)
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
