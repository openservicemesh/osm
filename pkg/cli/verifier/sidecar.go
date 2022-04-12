package verifier

import (
	"context"
	"fmt"
	"io"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"

	"github.com/openservicemesh/osm/pkg/constants"
)

// SidecarVerifier implements the Verifier interface for Envoy sidecar
type SidecarVerifier struct {
	stdout     io.Writer
	stderr     io.Writer
	kubeClient kubernetes.Interface
	pod        types.NamespacedName
}

// NewSidecarVerifier returns a Verifier for Envoy sidecar verification
func NewSidecarVerifier(stdout io.Writer, stderr io.Writer, kubeClient kubernetes.Interface, pod types.NamespacedName) Verifier {
	return &SidecarVerifier{
		stdout:     stdout,
		stderr:     stderr,
		kubeClient: kubeClient,
		pod:        pod,
	}
}

// Run executes the sidecar verifier
func (v *SidecarVerifier) Run() Result {
	result := Result{
		Context: fmt.Sprintf("Verify Envoy sidecar on pod %q", v.pod),
	}

	p, err := v.kubeClient.CoreV1().Pods(v.pod.Namespace).Get(context.Background(), v.pod.Name, metav1.GetOptions{})
	if err != nil {
		result.Status = Failure
		result.Reason = err.Error()
		result.Suggestion = fmt.Sprintf("Confirm pod %q exists", v.pod)
		return result
	}

	// Check if the Envoy sidecar is present
	foundEnvoy := false
	for _, container := range p.Spec.Containers {
		if container.Name == constants.EnvoyContainerName {
			foundEnvoy = true
			break
		}
	}
	if !foundEnvoy {
		result.Status = Failure
		result.Reason = fmt.Sprintf("Did not find Envoy sidecar on pod %q", v.pod)
		result.Suggestion = fmt.Sprintf("Ensure pod %q has sidecar injection enabled", v.pod)
		return result
	}

	result.Status = Success
	return result
}
