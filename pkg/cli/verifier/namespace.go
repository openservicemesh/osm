package verifier

import (
	"context"
	"fmt"
	"io"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/openservicemesh/osm/pkg/constants"
)

// NamespaceMonitorVerifier implements the Verifier interface for pod connectivity
type NamespaceMonitorVerifier struct {
	stdout     io.Writer
	stderr     io.Writer
	kubeClient kubernetes.Interface
	namespace  string
	meshName   string
}

// NewNamespaceMonitorVerifier implements verification for namespace monitoring
func NewNamespaceMonitorVerifier(stdout io.Writer, stderr io.Writer, kubeClient kubernetes.Interface, namespace string, meshName string) Verifier {
	return &NamespaceMonitorVerifier{
		stdout:     stdout,
		stderr:     stderr,
		kubeClient: kubeClient,
		namespace:  namespace,
		meshName:   meshName,
	}
}

// Run executes the namespace monitor verification
func (v *NamespaceMonitorVerifier) Run() Result {
	result := Result{
		Context: fmt.Sprintf("Verify if namespace %q is monitored", v.namespace),
	}

	ns, err := v.kubeClient.CoreV1().Namespaces().Get(context.Background(), v.namespace, metav1.GetOptions{})
	if err != nil {
		result.Status = Failure
		result.Reason = fmt.Sprintf("Error fetching namespace %q", v.namespace)
		return result
	}

	annotatedMeshName, ok := ns.Labels[constants.OSMKubeResourceMonitorAnnotation]
	if !ok {
		result.Status = Failure
		result.Reason = fmt.Sprintf("Missing label %q on namespace %q", constants.OSMKubeResourceMonitorAnnotation, v.namespace)
		result.Suggestion = fmt.Sprintf("Add label %q on namespace %q to include it in the mesh and restart the app", constants.OSMKubeResourceMonitorAnnotation, v.namespace)
		return result
	}
	if annotatedMeshName != v.meshName {
		result.Status = Failure
		result.Reason = fmt.Sprintf("Expected label %q to have value %q, got %q",
			constants.OSMKubeResourceMonitorAnnotation, v.meshName, annotatedMeshName)
		return result
	}

	result.Status = Success
	return result
}
