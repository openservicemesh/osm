package main

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"github.com/golang/glog"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	namespace := "smc"
	podName := "ads"
	labelSelector := "app=bookbuyer"
	clientset := getClient()

	podList, err := clientset.CoreV1().Pods(namespace).List(metav1.ListOptions{LabelSelector: labelSelector})
	if err != nil {
		glog.Error("Error fetching pods: ", err)
		os.Exit(1)
	}

	if podlen(podList) == 0 {
		glog.Error("Zero pods found")
		os.Exit(1)
	}

	req := clientset.CoreV1().Pods(namespace).GetLogs(podName, &v1.PodLogOptions{})
	podLogs, err := req.Stream()
	if err != nil {
		fmt.Println("error in opening stream")
	}
	defer podLogs.Close()

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, podLogs)
	if err != nil {
		fmt.Println("error in copy information from podLogs to buf")
	}
	str := buf.String()
	fmt.Println(str)
}

func getClient() *kubernetes.Clientset {
	var kubeConfig *rest.Config
	var err error
	kubeConfigFile := os.Getenv("KUBECONFIG")
	if kubeConfigFile != "" {
		kubeConfig, err = clientcmd.BuildConfigFromFlags("", kubeConfigFile)
		if err != nil {
			glog.Fatalf("Error fetching Kubernetes config. Ensure correctness of CLI argument 'kubeconfig=%s': %s", kubeConfigFile, err)
		}
	} else {
		// creates the in-cluster config
		kubeConfig, err = rest.InClusterConfig()
		if err != nil {
			glog.Fatalf("Error generating Kubernetes config: %s", err)
		}
	}

	clientset, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		fmt.Println("error in getting access to K8S")
		os.Exit(1)
	}
	return clientset
}

func (clientset *Clientset) GetPods(namespace string, labelSelector string) ([]corev1.Pod, error) {

	if err != nil {
		return []corev1.Pod{}, err
	}
	return podList.Items, nil
}
