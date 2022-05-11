package verifier

import (
	"fmt"
	"io"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	configv1alpha2 "github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"
)

// EgressConnectivityVerifier implements the Verifier interface for egress connectivity
type EgressConnectivityVerifier struct {
	stdout             io.Writer
	stderr             io.Writer
	kubeClient         kubernetes.Interface
	meshConfig         *configv1alpha2.MeshConfig
	trafficAttr        TrafficAttribute
	srcPodConfigGetter ConfigGetter
	meshName           string
}

// NewEgressConnectivityVerifier implements verification for egress connectivity
func NewEgressConnectivityVerifier(stdout io.Writer, stderr io.Writer, restConfig *rest.Config, kubeClient kubernetes.Interface,
	meshConfig *configv1alpha2.MeshConfig, trafficAttr TrafficAttribute,
	meshName string) Verifier {
	return &EgressConnectivityVerifier{
		stdout:      stdout,
		stderr:      stderr,
		kubeClient:  kubeClient,
		meshConfig:  meshConfig,
		trafficAttr: trafficAttr,
		srcPodConfigGetter: &PodConfigGetter{
			restConfig: restConfig,
			kubeClient: kubeClient,
			pod:        *trafficAttr.SrcPod,
		},
		meshName: meshName,
	}
}

// Run executes the pod connectivity verifier
func (v *EgressConnectivityVerifier) Run() Result {
	ctx := fmt.Sprintf("Verify if pod %q can access external service on port %d", v.trafficAttr.SrcPod, v.trafficAttr.ExternalPort)

	verifiers := Set{
		//
		// Namespace monitor verification
		NewNamespaceMonitorVerifier(v.stdout, v.stderr, v.kubeClient, v.trafficAttr.SrcPod.Namespace, v.meshName),
		//
		// Envoy sidecar verification
		NewSidecarVerifier(v.stdout, v.stderr, v.kubeClient, *v.trafficAttr.SrcPod),
		//
		// Envoy config verification
		NewEnvoyConfigVerifier(v.stdout, v.stderr, v.kubeClient, v.meshConfig, configAttribute{
			trafficAttr:     v.trafficAttr,
			srcConfigGetter: v.srcPodConfigGetter,
		}),
	}

	return verifiers.Run(ctx)
}
