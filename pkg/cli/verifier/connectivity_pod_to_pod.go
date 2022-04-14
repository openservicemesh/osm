package verifier

import (
	"fmt"
	"io"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

// PodConnectivityVerifier implements the Verifier interface for pod connectivity
type PodConnectivityVerifier struct {
	stdout      io.Writer
	stderr      io.Writer
	kubeClient  kubernetes.Interface
	srcPod      types.NamespacedName
	dstPod      types.NamespacedName
	appProtocol string
	meshName    string
}

// NewPodConnectivityVerifier implements verification for pod connectivity
func NewPodConnectivityVerifier(stdout io.Writer, stderr io.Writer, kubeClient kubernetes.Interface,
	srcPod types.NamespacedName, dstPod types.NamespacedName, appProtocol string, meshName string) Verifier {
	return &PodConnectivityVerifier{
		stdout:      stdout,
		stderr:      stderr,
		kubeClient:  kubeClient,
		srcPod:      srcPod,
		dstPod:      dstPod,
		appProtocol: appProtocol,
		meshName:    meshName,
	}
}

// Run executes the pod connectivity verifier
func (v *PodConnectivityVerifier) Run() Result {
	ctx := fmt.Sprintf("Verify if pod %q can access pod %q for app protocol %q", v.srcPod, v.dstPod, v.appProtocol)

	verifiers := Set{
		// ---
		// Verify prerequisites
		//
		// Namespace monitor verification
		NewNamespaceMonitorVerifier(v.stdout, v.stderr, v.kubeClient, v.srcPod.Namespace, v.meshName),
		NewNamespaceMonitorVerifier(v.stdout, v.stderr, v.kubeClient, v.dstPod.Namespace, v.meshName),
	}

	return verifiers.Run(ctx)
}
