package tests

import (
	"context"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// GetUnique gets a slice of strings and returns a slice with the unique strings
func GetUnique(slice []string) []string {
	// Map as a set
	uniqueSet := make(map[string]struct{})
	uniqueList := []string{}

	for _, item := range slice {
		uniqueSet[item] = struct{}{}
	}

	for keyName := range uniqueSet {
		uniqueList = append(uniqueList, keyName)
	}

	return uniqueList
}

// MakeService creates a new service for a set of pods with matching selectors
func MakeService(kubeClient kubernetes.Interface, svcName string, selectors map[string]string) (*v1.Service, error) {
	service := NewServiceFixture(svcName, Namespace, selectors)
	createdService, err := kubeClient.CoreV1().Services(Namespace).Create(context.TODO(), service, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}
	return createdService, nil
}

// MakePod creates a pod
func MakePod(kubeClient kubernetes.Interface, namespace, podName, serviceAccountName string, labels map[string]string) (*v1.Pod, error) {
	requestedPod := NewPodFixture(namespace, podName, serviceAccountName, labels)
	createdPod, err := kubeClient.CoreV1().Pods(namespace).Create(context.TODO(), &requestedPod, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}
	return createdPod, nil
}
