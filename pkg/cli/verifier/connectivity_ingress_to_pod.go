package verifier

import (
	"fmt"
	"io"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	configv1alpha2 "github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"
	"github.com/openservicemesh/osm/pkg/constants"
	policyClientset "github.com/openservicemesh/osm/pkg/gen/client/policy/clientset/versioned"
)

type IngressConnectivityVerifier struct {
	stdout             io.Writer
	stderr             io.Writer
	kubeClient         kubernetes.Interface
	policyClient       policyClientset.Interface
	meshConfig         *configv1alpha2.MeshConfig
	trafficAttr        TrafficAttribute
	dstPodConfigGetter ConfigGetter
	meshName           string
}

// NewIngressConnectivityVerifier implements verification for pod connectivity
func NewIngressConnectivityVerifier(stdout io.Writer, stderr io.Writer, restConfig *rest.Config, kubeClient kubernetes.Interface,
	policyClient policyClientset.Interface, meshConfig *configv1alpha2.MeshConfig, trafficAttr TrafficAttribute,
	meshName string) Verifier {
	return &IngressConnectivityVerifier{
		stdout:       stdout,
		stderr:       stderr,
		kubeClient:   kubeClient,
		policyClient: policyClient,
		meshConfig:   meshConfig,
		trafficAttr:  trafficAttr,
		dstPodConfigGetter: &PodConfigGetter{
			restConfig: restConfig,
			kubeClient: kubeClient,
			pod:        *trafficAttr.DstPod,
		},
		meshName: meshName,
	}
}

// Run executes the pod connectivity verifier
func (v *IngressConnectivityVerifier) Run() Result {
	ctx := fmt.Sprintf("Verify if pod %q can access pod %q for app protocol %q via ingress", v.trafficAttr.SrcPod, v.trafficAttr.DstPod, v.trafficAttr.AppProtocol)
	if v.trafficAttr.AppProtocol != constants.ProtocolHTTP {
		// We're not going to deal with mTLS yet
		return Result{
			Context: ctx,
			Status:  Failure,
			Reason:  fmt.Sprintf("Ingress verifier only supports the following protocol(s): %q", v.trafficAttr.AppProtocol),
		}
	}

	verifiers := Set{
		// Namespace monitor verification
		NewNamespaceMonitorVerifier(v.stdout, v.stderr, v.kubeClient, v.trafficAttr.SrcPod.Namespace, v.meshName),
		NewNamespaceMonitorVerifier(v.stdout, v.stderr, v.kubeClient, v.trafficAttr.DstPod.Namespace, v.meshName),

		// Envoy sidecar verification
		NewSidecarVerifier(v.stdout, v.stderr, v.kubeClient, *v.trafficAttr.SrcPod, WithVerifyAbsence()), // ingress pod shouldn't have a sidecar
		NewSidecarVerifier(v.stdout, v.stderr, v.kubeClient, *v.trafficAttr.DstPod),

		// IngressBackend verification
		NewIngressBackendVerifier(v.stdout, v.stderr, v.policyClient, v.trafficAttr.AppProtocol, *v.trafficAttr.IngressBackend, *v.trafficAttr.SrcService),

		// Envoy config verification
		NewEnvoyConfigVerifier(v.stdout, v.stderr, v.kubeClient, v.meshConfig, configAttribute{
			trafficAttr:     v.trafficAttr,
			dstConfigGetter: v.dstPodConfigGetter,
		}),

		// TODO: Verify there are no IngressBackend resources referencing the same backend
	}

	return verifiers.Run(ctx)
}
