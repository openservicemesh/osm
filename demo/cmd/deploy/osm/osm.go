package osm

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// DeployOSM deploys various components of OSM service
func DeployOSM(clientSet *kubernetes.Clientset, namespace string) error {
	if err := deployOSMService(clientSet, namespace); err != nil {
		return fmt.Errorf("Unable to deploy osm service : %v", err)
	}
	if err := applyOSMDeployment(clientSet, namespace); err != nil {
		return fmt.Errorf("Unable to deploy osm pod : %v", err)
	}
	return nil
}

func deployOSMService(clientSet *kubernetes.Clientset, namespace string) error {
	service := generateOSMService(namespace)
	if _, err := clientSet.CoreV1().Services(namespace).Create(context.Background(), service, metav1.CreateOptions{}); err != nil {
		return err
	}
	return nil
}

func applyOSMDeployment(clientSet *kubernetes.Clientset, namespace string) error {
	deployment := generateOSMDeployment(namespace)
	if _, err := clientSet.AppsV1().Deployments(namespace).Create(context.Background(), deployment, metav1.CreateOptions{}); err != nil {
		return err
	}
	return nil
}
