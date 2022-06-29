package verifier

import (
	"io"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/openservicemesh/osm/pkg/constants"
)

// ControlPlaneHealthVerifier implements the Verifier interface for control plane health
type ControlPlaneHealthVerifier struct {
	stdout           io.Writer
	stderr           io.Writer
	kubeClient       kubernetes.Interface
	restConfig       *rest.Config
	namespace        string
	controllerProber httpProber
	injectorProber   httpProber
	bootstrapProber  httpProber
}

// NewControlPlaneHealthVerifier implements verification for control plane health
func NewControlPlaneHealthVerifier(stdout io.Writer, stderr io.Writer, kubeClient kubernetes.Interface, restConfig *rest.Config, namespace string) Verifier {
	return &ControlPlaneHealthVerifier{
		stdout:     stdout,
		stderr:     stderr,
		kubeClient: kubeClient,
		restConfig: restConfig,
		namespace:  namespace,
		controllerProber: &podProber{
			kubeClient: kubeClient,
			restConfig: restConfig,
			port:       constants.OSMHTTPServerPort,
			path:       constants.OSMControllerLivenessPath,
			protocol:   constants.ProtocolHTTP,
		},
		injectorProber: &podProber{
			kubeClient: kubeClient,
			restConfig: restConfig,
			port:       constants.OSMHTTPServerPort,
			path:       constants.WebhookHealthPath,
			protocol:   constants.ProtocolHTTP,
		},
		bootstrapProber: &podProber{
			kubeClient: kubeClient,
			restConfig: restConfig,
			port:       constants.OSMHTTPServerPort,
			path:       constants.WebhookHealthPath,
			protocol:   constants.ProtocolHTTP,
		},
	}
}

// Run executes the control plane health verifier
func (v *ControlPlaneHealthVerifier) Run() Result {
	ctx := "Verify the health of OSM control plane"

	verifiers := Set{
		// Pod readiness verification
		NewPodStatusVerifier(v.stdout, v.stderr, v.kubeClient, types.NamespacedName{Namespace: v.namespace, Name: constants.OSMControllerName}),
		NewPodStatusVerifier(v.stdout, v.stderr, v.kubeClient, types.NamespacedName{Namespace: v.namespace, Name: constants.OSMInjectorName}),
		NewPodStatusVerifier(v.stdout, v.stderr, v.kubeClient, types.NamespacedName{Namespace: v.namespace, Name: constants.OSMBootstrapName}),

		// Pod liveness health probe verification
		NewPodProbeVerifier(v.stdout, v.stderr, v.kubeClient,
			types.NamespacedName{Namespace: v.namespace, Name: constants.OSMControllerName}, v.controllerProber),
		NewPodProbeVerifier(v.stdout, v.stderr, v.kubeClient,
			types.NamespacedName{Namespace: v.namespace, Name: constants.OSMInjectorName}, v.injectorProber),
		NewPodProbeVerifier(v.stdout, v.stderr, v.kubeClient,
			types.NamespacedName{Namespace: v.namespace, Name: constants.OSMBootstrapName}, v.bootstrapProber),
	}

	return verifiers.Run(ctx)
}
