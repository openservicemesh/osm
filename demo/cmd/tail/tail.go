package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/deislabs/smc/demo/cmd/common"
	"github.com/golang/glog"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	waitTime = 1 * time.Second
)

func main() {
	// Log output is buffered... Calling Flush before exiting guarantees all log output is written.
	defer glog.Flush()

	// Workaround for "ERROR: logging before flag.Parse"
	// See: https://github.com/kubernetes/kubernetes/issues/17162#issuecomment-225596212
	_ = flag.CommandLine.Parse([]string{})

	namespace := "smc"
	containerName := "bookbuyer"
	labelSelector := "app=bookbuyer"

	clientset := getClient()

	podList, err := clientset.CoreV1().Pods(namespace).List(metav1.ListOptions{LabelSelector: labelSelector})
	if err != nil {
		glog.Fatal("Error fetching pods: ", err)
	}

	if len(podList.Items) == 0 {
		glog.Fatal("Zero pods found")
	}

	sort.SliceStable(podList.Items, func(i, j int) bool {
		p1 := podList.Items[i].CreationTimestamp.UnixNano()
		p2 := podList.Items[j].CreationTimestamp.UnixNano()
		return p1 > p2
	})

	for idx, pod := range podList.Items {
		glog.Infof("%d: %s -> %s\n", idx, pod.Name, pod.CreationTimestamp)
	}

	podName := podList.Items[0].Name
Run:
	for {
		if pod, err := clientset.CoreV1().Pods(namespace).Get(podName, metav1.GetOptions{}); err != nil {
			glog.Infof("Error getting pod %s/%s: %s", namespace, podName, err)
		} else {
			for _, container := range pod.Status.ContainerStatuses {
				if container.State.Waiting != nil && container.State.Waiting.Reason == "PodInitializing" {
					glog.Info("Pod %s/%s is still initializing; Waiting %+v", namespace, podName, waitTime)
					time.Sleep(waitTime)
				} else {
					break Run
				}
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
		glog.Fatal("Error in opening stream: ", err)
	}
	defer podLogs.Close()

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, podLogs)
	if err != nil {
		glog.Infoln("error in copy information from podLogs to buf")
	}
	logs := buf.String()
	if strings.HasSuffix(logs, common.Success) {
		glog.Info("The test succeeded")
		os.Exit(0)
	}
	fmt.Print(logs)
	os.Exit(1)
}

func getClient() *kubernetes.Clientset {
	var kubeConfig *rest.Config
	var err error
	kubeConfigFile := os.Getenv("KUBECONFIG")
	if kubeConfigFile != "" {
		kubeConfig, err = clientcmd.BuildConfigFromFlags("", kubeConfigFile)
		if err != nil {
			glog.Infof("Error fetching Kubernetes config. Ensure correctness of CLI argument 'kubeconfig=%s': %s", kubeConfigFile, err)
			os.Exit(1)
		}
	} else {
		// creates the in-cluster config
		kubeConfig, err = rest.InClusterConfig()
		if err != nil {
			glog.Infof("Error generating Kubernetes config: %s", err)
			os.Exit(1)
		}
	}

	clientset, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		glog.Infoln("error in getting access to K8S")
		os.Exit(1)
	}
	return clientset
}
