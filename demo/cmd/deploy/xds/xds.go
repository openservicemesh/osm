package xds

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// DeployXDS deploys various compontents of xds service
func DeployXDS(clientSet *kubernetes.Clientset, namespace string) error {
	if err := deployXDSService(clientSet, namespace); err != nil {
		return fmt.Errorf("Unable to deploy xds service : %v", err)
	}
	if err := deployXDSPod(clientSet, namespace); err != nil {
		return fmt.Errorf("Unable to deploy xds pod : %v", err)
	}
	return nil
}

func deployXDSService(clientSet *kubernetes.Clientset, namespace string) error {
	service := generateXDSService(namespace)
	if _, err := clientSet.CoreV1().Services(namespace).Create(context.Background(), service, metav1.CreateOptions{}); err != nil {
		return err
	}
	return nil
}

func deployXDSPod(clientSet *kubernetes.Clientset, namespace string) error {
	pod := generateXDSPod(namespace)
	if _, err := clientSet.CoreV1().Pods(namespace).Create(context.Background(), pod, metav1.CreateOptions{}); err != nil {
		return err
	}
	return nil
}
