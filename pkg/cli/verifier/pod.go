package verifier

import (
	"context"
	"fmt"
	"io"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"

	"github.com/openservicemesh/osm/pkg/constants"
)

// PodStatusVerifier implements the Verifier interface for control plane health
type PodStatusVerifier struct {
	stdout     io.Writer
	stderr     io.Writer
	kubeClient kubernetes.Interface
	app        types.NamespacedName
}

// NewPodStatusVerifier implements verification for control plane health
func NewPodStatusVerifier(stdout io.Writer, stderr io.Writer, kubeClient kubernetes.Interface, app types.NamespacedName) Verifier {
	return &PodStatusVerifier{
		stdout:     stdout,
		stderr:     stderr,
		kubeClient: kubeClient,
		app:        app,
	}
}

// Run executes the pod status verifier
func (v *PodStatusVerifier) Run() Result {
	result := Result{
		Context: fmt.Sprintf("Verify status of pod for app %s", v.app),
	}

	labelSelector := metav1.LabelSelector{MatchLabels: map[string]string{constants.AppLabel: v.app.Name}}
	listOptions := metav1.ListOptions{
		LabelSelector: labels.Set(labelSelector.MatchLabels).String(),
	}
	podList, err := v.kubeClient.CoreV1().Pods(v.app.Namespace).List(context.TODO(), listOptions)
	if err != nil || podList == nil {
		result.Status = Unknown
		result.Reason = fmt.Sprintf("Error fetching pods for app %s, err: %s", v.app, err)
		return result
	}

	if len(podList.Items) == 0 {
		result.Status = Failure
		result.Reason = fmt.Sprintf("No pods found for app %s", v.app)
		return result
	}

	var failureReasons []string
	for _, pod := range podList.Items {
		for _, cond := range pod.Status.Conditions {
			if cond.Type == corev1.PodScheduled && cond.Status != corev1.ConditionTrue {
				failureReasons = append(failureReasons, fmt.Sprintf("%s/%s not scheduled", pod.Namespace, pod.Name))
				break
			}
			if cond.Type == corev1.ContainersReady && cond.Status != corev1.ConditionTrue {
				failureReasons = append(failureReasons, fmt.Sprintf("%s/%s containers not ready", pod.Namespace, pod.Name))
				break
			}
			if cond.Type == corev1.PodInitialized && cond.Status != corev1.ConditionTrue {
				failureReasons = append(failureReasons, fmt.Sprintf("%s/%s init-containers pending completion", pod.Namespace, pod.Name))
				break
			}
			if cond.Type == corev1.PodReady && cond.Status != corev1.ConditionTrue {
				failureReasons = append(failureReasons, fmt.Sprintf("%s/%s not ready", pod.Namespace, pod.Name))
			}
		}
	}
	if len(failureReasons) > 0 {
		result.Status = Failure
		result.Reason = fmt.Sprintf("Pod not ready: %s", strings.Join(failureReasons, ", "))
		return result
	}

	result.Status = Success
	return result
}
