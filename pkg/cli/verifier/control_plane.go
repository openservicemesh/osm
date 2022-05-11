package verifier

import (
	"io"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"

	"github.com/openservicemesh/osm/pkg/constants"
)

// ControlPlaneHealthVerifier implements the Verifier interface for control plane health
type ControlPlaneHealthVerifier struct {
	stdout     io.Writer
	stderr     io.Writer
	kubeClient kubernetes.Interface
	namespace  string
}

// NewControlPlaneHealthVerifier implements verification for control plane health
func NewControlPlaneHealthVerifier(stdout io.Writer, stderr io.Writer, kubeClient kubernetes.Interface, namespace string) Verifier {
	return &ControlPlaneHealthVerifier{
		stdout:     stdout,
		stderr:     stderr,
		kubeClient: kubeClient,
		namespace:  namespace,
	}
}

// Run executes the control plane health verifier
func (v *ControlPlaneHealthVerifier) Run() Result {
	ctx := "Verify the health of OSM control plane"

	verifiers := Set{
		// Pod status verification
		NewPodStatusVerifier(v.stdout, v.stderr, v.kubeClient, types.NamespacedName{Namespace: v.namespace, Name: constants.OSMControllerName}),
		NewPodStatusVerifier(v.stdout, v.stderr, v.kubeClient, types.NamespacedName{Namespace: v.namespace, Name: constants.OSMInjectorName}),
		NewPodStatusVerifier(v.stdout, v.stderr, v.kubeClient, types.NamespacedName{Namespace: v.namespace, Name: constants.OSMBootstrapName}),
	}

	return verifiers.Run(ctx)
}
