package xds

import (
	"fmt"

	"k8s.io/client-go/kubernetes"
)

// Xds deploys xds service
func Xds(clientSet *kubernetes.Clientset, namespace string) error {
	if err := deployXdsService(clientSet, namespace); err != nil {
		return fmt.Errorf("Unable to deploy xds service : %v", err)
	}
	if err := deployXdsPod(clientSet, namespace); err != nil {
		return fmt.Errorf("Unable to deploy xds pod : %v", err)
	}
	return nil
}

func deployXdsService(clientSet *kubernetes.Clientset, namespace string) error {
	service := generateXdsService(namespace)
	if _, err := clientSet.CoreV1().Services(namespace).Create(service); err != nil {
		return err
	}
	return nil
}

func deployXdsPod(clientSet *kubernetes.Clientset, namespace string) error {
	pod := generateXdsPod(namespace)
	if _, err := clientSet.CoreV1().Pods(namespace).Create(pod); err != nil {
		return err
	}
	return nil
}
