package verifier

import (
	"context"
	"fmt"
	"io"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	configv1alpha2 "github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"
	"github.com/openservicemesh/osm/pkg/constants"
	policyClientset "github.com/openservicemesh/osm/pkg/gen/client/policy/clientset/versioned"
)

// IngressConnectivityVerifier implements verification for pod connectivity
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

// NewIngressConnectivityVerifier creates a new IngressConnectivityVerifier
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
	ctx := fmt.Sprintf("Verify if service %q can access pod %q for app protocol %q via ingress", v.trafficAttr.SrcService, v.trafficAttr.DstPod, v.trafficAttr.AppProtocol)
	if v.trafficAttr.AppProtocol != constants.ProtocolHTTP {
		// TODO: Handle HTTPS
		return Result{
			Context: ctx,
			Status:  Failure,
			Reason:  fmt.Sprintf("Ingress verifier only supports the following protocol(s): %q", v.trafficAttr.AppProtocol),
		}
	}

	srcService, err := v.kubeClient.CoreV1().Services(v.trafficAttr.SrcService.Namespace).Get(context.Background(), v.trafficAttr.SrcService.Name, v1.GetOptions{})
	if err != nil {
		return Result{
			Context: ctx,
			Status:  Failure,
			Reason:  fmt.Sprintf("Error retrieving source ingress service %s: %s", v.trafficAttr.SrcService, err),
		}
	}

	selector, _ := labels.ValidatedSelectorFromSet(srcService.Spec.Selector)
	pods, err := v.kubeClient.CoreV1().Pods(v.trafficAttr.SrcService.Namespace).List(context.Background(), v1.ListOptions{
		LabelSelector: selector.String(),
	})

	if err != nil {
		return Result{
			Context: ctx,
			Status:  Failure,
			Reason:  fmt.Sprintf("Error retrieving pods from source ingress service %s: %s", v.trafficAttr.SrcService, err),
		}
	}

	verifiers := Set{
		// Namespace monitor verification
		NewNamespaceMonitorVerifier(v.stdout, v.stderr, v.kubeClient, v.trafficAttr.SrcService.Namespace, v.meshName),
		NewNamespaceMonitorVerifier(v.stdout, v.stderr, v.kubeClient, v.trafficAttr.DstPod.Namespace, v.meshName),
	}

	for _, pod := range pods.Items {
		verifiers = append(verifiers, NewSidecarVerifier(v.stdout, v.stderr, v.kubeClient, types.NamespacedName{
			Namespace: pod.ObjectMeta.Namespace,
			Name:      pod.ObjectMeta.Name,
		}, WithVerifyAbsence())) // ingress pods shouldn't have a sidecar
	}

	verifiers = append(verifiers, Set{
		// Envoy sidecar verification

		NewSidecarVerifier(v.stdout, v.stderr, v.kubeClient, *v.trafficAttr.DstPod),

		// IngressBackend verification
		NewIngressBackendVerifier(v.stdout, v.stderr, v.policyClient, v.trafficAttr.AppProtocol, v.trafficAttr.DstPort, *v.trafficAttr.IngressBackend, *v.trafficAttr.SrcService),

		// Envoy config verification
		NewEnvoyConfigVerifier(v.stdout, v.stderr, v.kubeClient, v.meshConfig, configAttribute{
			trafficAttr:     v.trafficAttr,
			dstConfigGetter: v.dstPodConfigGetter,
		}),
	}...)

	return verifiers.Run(ctx)
}
