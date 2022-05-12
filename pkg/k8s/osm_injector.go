package k8s

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"

	"github.com/openservicemesh/osm/pkg/constants"
)

// GetOSMInjectorPods returns a list of osm-injector pods in the namespace
func GetOSMInjectorPods(clientSet kubernetes.Interface, ns string) *corev1.PodList {
	labelSelector := metav1.LabelSelector{MatchLabels: map[string]string{constants.AppLabel: constants.OSMInjectorName}}
	listOptions := metav1.ListOptions{
		LabelSelector: labels.Set(labelSelector.MatchLabels).String(),
	}
	podList, _ := clientSet.CoreV1().Pods(ns).List(context.TODO(), listOptions)
	return podList
}
