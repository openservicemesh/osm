package verifier

import (
	"fmt"
	"io"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	configv1alpha2 "github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"
)

// TrafficAttribute describes the attributes of the traffic
type TrafficAttribute struct {
	SrcPod         *types.NamespacedName
	SrcService     *types.NamespacedName
	DstPod         *types.NamespacedName
	DstService     *types.NamespacedName
	IngressBackend *types.NamespacedName
	DstPort        uint16
	ExternalHost   string
	ExternalPort   uint16
	AppProtocol    string
	IsIngress      bool
}

// PodConnectivityVerifier implements the Verifier interface for pod connectivity
type PodConnectivityVerifier struct {
	stdout             io.Writer
	stderr             io.Writer
	kubeClient         kubernetes.Interface
	meshConfig         *configv1alpha2.MeshConfig
	trafficAttr        TrafficAttribute
	srcPodConfigGetter ConfigGetter
	dstPodConfigGetter ConfigGetter
	meshName           string
}

// NewPodConnectivityVerifier implements verification for pod connectivity
func NewPodConnectivityVerifier(stdout io.Writer, stderr io.Writer, restConfig *rest.Config, kubeClient kubernetes.Interface,
	meshConfig *configv1alpha2.MeshConfig, trafficAttr TrafficAttribute,
	meshName string) Verifier {
	return &PodConnectivityVerifier{
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
		dstPodConfigGetter: &PodConfigGetter{
			restConfig: restConfig,
			kubeClient: kubeClient,
			pod:        *trafficAttr.DstPod,
		},
		meshName: meshName,
	}
}

// Run executes the pod connectivity verifier
func (v *PodConnectivityVerifier) Run() Result {
	ctx := fmt.Sprintf("Verify if pod %q can access pod %q for service %q", v.trafficAttr.SrcPod, v.trafficAttr.DstPod, v.trafficAttr.DstService)

	verifiers := Set{
		//
		// Namespace monitor verification
		NewNamespaceMonitorVerifier(v.stdout, v.stderr, v.kubeClient, v.trafficAttr.SrcPod.Namespace, v.meshName),
		NewNamespaceMonitorVerifier(v.stdout, v.stderr, v.kubeClient, v.trafficAttr.DstPod.Namespace, v.meshName),
		//
		// Envoy sidecar verification
		NewSidecarVerifier(v.stdout, v.stderr, v.kubeClient, *v.trafficAttr.SrcPod),
		NewSidecarVerifier(v.stdout, v.stderr, v.kubeClient, *v.trafficAttr.DstPod),
		//
		// Envoy config verification
		NewEnvoyConfigVerifier(v.stdout, v.stderr, v.kubeClient, v.meshConfig, configAttribute{
			trafficAttr:     v.trafficAttr,
			srcConfigGetter: v.srcPodConfigGetter,
			dstConfigGetter: v.dstPodConfigGetter,
		}),
	}

	return verifiers.Run(ctx)
}
