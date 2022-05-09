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
	stdout        io.Writer
	stderr        io.Writer
	kubeClient    kubernetes.Interface
	pod           types.NamespacedName
	verifyAbsence bool
}

// SidecarVerifierOpt exposes a way to modify the behavior of the sidecar verifier without modifying the base contract
type SidecarVerifierOpt func(*SidecarVerifier)

// NewSidecarVerifier returns a Verifier for Envoy sidecar verification
func NewSidecarVerifier(stdout io.Writer, stderr io.Writer, kubeClient kubernetes.Interface, pod types.NamespacedName, opts ...SidecarVerifierOpt) Verifier {
	baseVerifier := &SidecarVerifier{
		stdout:     stdout,
		stderr:     stderr,
		kubeClient: kubeClient,
		pod:        pod,
	}

	for _, opt := range opts {
		opt(baseVerifier)
	}

	return baseVerifier
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

	// we found a sidecar when one wasn't expected (e.g. for ingress)
	if foundEnvoy && v.verifyAbsence {
		result.Status = Failure
		result.Reason = fmt.Sprintf("Found Envoy sidecar on pod %q when one wasn't expected", v.pod)
		result.Suggestion = fmt.Sprintf("Ensure pod %q does not have sidecar injection enabled", v.pod)
		return result
	}

	// we did not find a sidecar when one was expected (base case)
	if !foundEnvoy && !v.verifyAbsence {
		result.Status = Failure
		result.Reason = fmt.Sprintf("Did not find Envoy sidecar on pod %q", v.pod)
		result.Suggestion = fmt.Sprintf("Ensure pod %q has sidecar injection enabled", v.pod)
		return result
	}

	result.Status = Success
	return result
}

// WithVerifyAbsence sets verifyAbsence to true
func WithVerifyAbsence() SidecarVerifierOpt {
	return func(sv *SidecarVerifier) {
		sv.verifyAbsence = true
	}
}
