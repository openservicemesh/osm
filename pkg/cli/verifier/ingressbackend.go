package verifier

import (
	"context"
	"fmt"
	"io"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	policyClientset "github.com/openservicemesh/osm/pkg/gen/client/policy/clientset/versioned"
)

// IngressBackendVerifier implements the Verifier interface for ingressbackend verification
type IngressBackendVerifier struct {
	stdout          io.Writer
	stderr          io.Writer
	policyClient    policyClientset.Interface
	ingressBackend  types.NamespacedName
	ingressService  types.NamespacedName
	backendProtocol string
}

// NewIngressBackendVerifier creates a new IngressBackendVerifier
func NewIngressBackendVerifier(stdout io.Writer, stderr io.Writer, policyClient policyClientset.Interface, backendProtocol string, ingressBackend, ingressService types.NamespacedName) Verifier {
	return &IngressBackendVerifier{
		stdout:          stdout,
		stderr:          stderr,
		policyClient:    policyClient,
		ingressBackend:  ingressBackend,
		ingressService:  ingressService,
		backendProtocol: backendProtocol,
	}
}

// Run runs the IngressBackend verifier
func (v *IngressBackendVerifier) Run() Result {
	result := Result{
		Context: fmt.Sprintf("Verify IngressBackend %q", v.ingressBackend),
	}

	ib, err := v.policyClient.PolicyV1alpha1().IngressBackends(v.ingressBackend.Namespace).Get(context.Background(), v.ingressBackend.Name, metav1.GetOptions{})
	if err != nil {
		result.Status = Failure
		result.Reason = err.Error()
		result.Suggestion = fmt.Sprintf("Confirm IngressBackend %q exists", v.ingressBackend)
		return result
	}

	if len(ib.Spec.Backends) == 0 {
		result.Status = Failure
		result.Reason = fmt.Sprintf("No backends detected for IngressBackend %q", v.ingressBackend)
		result.Suggestion = fmt.Sprintf("Add a valid backend for IngressBackend %q", v.ingressBackend)
		return result
	}

	// TODO: check that IngressBackend port/protocol matches backendService port/protocol

	// No sources evaluates to a wildcard which shouldn't block anything
	if len(ib.Spec.Sources) == 0 {
		result.Status = Success
		result.Suggestion = "Allowing HTTP ingress without sources is insecure; use IngressBackend.Spec.Sources to restrict clients"
		return result
	}

	foundMatchingSource := false
	for _, src := range ib.Spec.Sources {
		if src.Kind == "Service" && src.Namespace == v.ingressService.Namespace && src.Name == v.ingressService.Name {
			foundMatchingSource = true
		}
	}

	if !foundMatchingSource {
		result.Status = Failure
		result.Reason = fmt.Sprintf("No source matching service %q found in IngressBackend %q", v.ingressService, v.ingressBackend)
		result.Suggestion = fmt.Sprintf("Add a source for service %q in IngressBackend %q", v.ingressService, v.ingressBackend)
		return result
	}

	result.Status = Success
	return result
}
