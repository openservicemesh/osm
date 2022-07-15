package injector

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"gomodules.xyz/jsonpatch/v2"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/errcode"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/metricsstore"
)

func (wh *mutatingWebhook) createPatch(pod *corev1.Pod, req *admissionv1.AdmissionRequest, proxyUUID uuid.UUID) ([]byte, error) {
	namespace := req.Namespace

	// Issue a certificate for the proxy sidecar - used for Envoy to connect to XDS (not Envoy-to-Envoy connections)
	cnPrefix := envoy.NewXDSCertCNPrefix(proxyUUID, envoy.KindSidecar, identity.New(pod.Spec.ServiceAccountName, namespace))
	log.Debug().Msgf("Patching POD spec: service-account=%s, namespace=%s with certificate CN prefix=%s", pod.Spec.ServiceAccountName, namespace, cnPrefix)
	startTime := time.Now()
	bootstrapCertificate, err := wh.certManager.IssueCertificate(cnPrefix, certificate.Internal)
	if err != nil {
		log.Error().Err(err).Msgf("Error issuing bootstrap certificate for Envoy with CN prefix=%s", cnPrefix)
		return nil, err
	}
	elapsed := time.Since(startTime)

	metricsstore.DefaultMetricsStore.CertIssuedCount.Inc()
	metricsstore.DefaultMetricsStore.CertIssuedTime.
		WithLabelValues().Observe(elapsed.Seconds())
	originalHealthProbes := rewriteHealthProbes(pod)

	// Create the bootstrap configuration for the Envoy proxy for the given pod
	envoyBootstrapConfigName := bootstrapSecretPrefix + proxyUUID.String()

	// This needs to occur before replacing the label below.
	originalUUID, alreadyInjected := getProxyUUID(pod)
	switch {
	case req.DryRun != nil && *req.DryRun:
		// The webhook has a side effect (making out-of-band changes) of creating k8s secret
		// corresponding to the Envoy bootstrap config. Such a side effect needs to be skipped
		// when the request is a DryRun.
		// Ref: https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/#side-effects
		log.Debug().Msgf("Skipping envoy bootstrap config creation for dry-run request: service-account=%s, namespace=%s", pod.Spec.ServiceAccountName, namespace)
	case alreadyInjected:
		// Pod definitions can be copied via the `kubectl debug` command, which can lead to a pod being created that
		// has already had injection occur. We could simply do nothing and return early, but that would leave 2 pods
		// with the same UUID, so instead we change the UUID, and create a new bootstrap config, copied from the original,
		// with the proxy UUID changed.
		oldConfigName := bootstrapSecretPrefix + originalUUID
		if _, err := wh.createEnvoyBootstrapFromExisting(envoyBootstrapConfigName, oldConfigName, namespace, bootstrapCertificate); err != nil {
			log.Error().Err(err).Msgf("Failed to create Envoy bootstrap config for already-injected pod: service-account=%s, namespace=%s, certificate CN prefix=%s", pod.Spec.ServiceAccountName, namespace, cnPrefix)
			return nil, err
		}
	default:
		if _, err = wh.createEnvoyBootstrapConfig(envoyBootstrapConfigName, namespace, wh.osmNamespace, bootstrapCertificate, originalHealthProbes); err != nil {
			log.Error().Err(err).Msgf("Failed to create Envoy bootstrap config for pod: service-account=%s, namespace=%s, certificate CN prefix=%s", pod.Spec.ServiceAccountName, namespace, cnPrefix)
			return nil, err
		}
	}
	enableMetrics, err := wh.isMetricsEnabled(namespace)
	if err != nil {
		log.Error().Err(err).Msgf("Error checking if namespace %s is enabled for metrics", namespace)
		return nil, err
	}
	if enableMetrics {
		if pod.Annotations == nil {
			pod.Annotations = make(map[string]string)
		}
		pod.Annotations[constants.PrometheusScrapeAnnotation] = strconv.FormatBool(true)
		pod.Annotations[constants.PrometheusPortAnnotation] = strconv.Itoa(constants.EnvoyPrometheusInboundListenerPort)
		pod.Annotations[constants.PrometheusPathAnnotation] = constants.PrometheusScrapePath
	}

	// This will append a label to the pod, which points to the unique Envoy ID used in the
	// xDS certificate for that Envoy. This label will help xDS match the actual pod to the Envoy that
	// connects to xDS (with the certificate's CN matching this label).
	if pod.Labels == nil {
		pod.Labels = make(map[string]string)
	}
	pod.Labels[constants.EnvoyUniqueIDLabelName] = proxyUUID.String()

	if alreadyInjected {
		// replace the volume and we're done.
		for i, volume := range pod.Spec.Volumes {
			// It should be the last, but we check all for posterity.
			if volume.Name == envoyBootstrapConfigVolume {
				pod.Spec.Volumes[i] = getVolumeSpec(envoyBootstrapConfigName)
				break
			}
		}
		return json.Marshal(makePatches(req, pod))
	}

	// Create volume for the envoy bootstrap config Secret
	pod.Spec.Volumes = append(pod.Spec.Volumes, getVolumeSpec(envoyBootstrapConfigName))

	// On Windows we cannot use init containers to program HNS because it requires elevated privileges
	// As a result we assume that the HNS redirection policies are already programmed via a CNI plugin.
	// Skip adding the init container and only patch the pod spec with sidecar container.
	podOS := pod.Spec.NodeSelector["kubernetes.io/os"]
	if err := wh.verifyPrerequisites(podOS); err != nil {
		return nil, err
	}

	err = wh.configurePodInit(podOS, pod, namespace)
	if err != nil {
		return nil, err
	}

	if originalHealthProbes.UsesTCP() {
		healthcheckContainer := corev1.Container{
			Name:            "osm-healthcheck",
			Image:           os.Getenv("OSM_DEFAULT_HEALTHCHECK_CONTAINER_IMAGE"),
			ImagePullPolicy: wh.osmContainerPullPolicy,
			Args: []string{
				"--verbosity", log.GetLevel().String(),
			},
			Command: []string{
				"/osm-healthcheck",
			},
			Ports: []corev1.ContainerPort{
				{
					ContainerPort: constants.HealthcheckPort,
				},
			},
		}
		pod.Spec.Containers = append(pod.Spec.Containers, healthcheckContainer)
	}

	// Add the Envoy sidecar
	sidecar := getEnvoySidecarContainerSpec(pod, wh.configurator, originalHealthProbes, podOS)
	pod.Spec.Containers = append(pod.Spec.Containers, sidecar)

	return json.Marshal(makePatches(req, pod))
}

// verifyPrerequisites verifies if the prerequisites to patch the request are met by returning an error if unmet
func (wh *mutatingWebhook) verifyPrerequisites(podOS string) error {
	isWindows := strings.EqualFold(podOS, constants.OSWindows)

	// Verify that the required images are configured
	if image := wh.configurator.GetEnvoyImage(); !isWindows && image == "" {
		// Linux pods require Envoy Linux image
		return fmt.Errorf("MeshConfig sidecar.envoyImage not set")
	}
	if image := wh.configurator.GetEnvoyWindowsImage(); isWindows && image == "" {
		// Windows pods require Envoy Windows image
		return fmt.Errorf("MeshConfig sidecar.envoyWindowsImage not set")
	}
	if image := wh.configurator.GetInitContainerImage(); !isWindows && image == "" {
		// Linux pods require init container image
		return fmt.Errorf("MeshConfig sidecar.initContainerImage not set")
	}

	return nil
}

func (wh *mutatingWebhook) configurePodInit(podOS string, pod *corev1.Pod, namespace string) error {
	if strings.EqualFold(podOS, constants.OSWindows) {
		// No init container for Windows
		return nil
	}

	// Build outbound port exclusion list
	podOutboundPortExclusionList, err := getPortExclusionListForPod(pod, namespace, outboundPortExclusionListAnnotation)
	if err != nil {
		return err
	}
	globalOutboundPortExclusionList := wh.configurator.GetMeshConfig().Spec.Traffic.OutboundPortExclusionList
	outboundPortExclusionList := mergePortExclusionLists(podOutboundPortExclusionList, globalOutboundPortExclusionList)

	// Build inbound port exclusion list
	podInboundPortExclusionList, err := getPortExclusionListForPod(pod, namespace, inboundPortExclusionListAnnotation)
	if err != nil {
		return err
	}
	globalInboundPortExclusionList := wh.configurator.GetMeshConfig().Spec.Traffic.InboundPortExclusionList
	inboundPortExclusionList := mergePortExclusionLists(podInboundPortExclusionList, globalInboundPortExclusionList)

	// Build the outbound IP range exclusion list
	podOutboundIPRangeExclusionList, err := getOutboundIPRangeListForPod(pod, namespace, outboundIPRangeExclusionListAnnotation)
	if err != nil {
		return err
	}
	globalOutboundIPRangeExclusionList := wh.configurator.GetMeshConfig().Spec.Traffic.OutboundIPRangeExclusionList
	outboundIPRangeExclusionList := mergeIPRangeLists(podOutboundIPRangeExclusionList, globalOutboundIPRangeExclusionList)

	// Build the outbound IP range inclusion list
	podOutboundIPRangeInclusionList, err := getOutboundIPRangeListForPod(pod, namespace, outboundIPRangeInclusionListAnnotation)
	if err != nil {
		return err
	}
	globalOutboundIPRangeInclusionList := wh.configurator.GetMeshConfig().Spec.Traffic.OutboundIPRangeInclusionList
	outboundIPRangeInclusionList := mergeIPRangeLists(podOutboundIPRangeInclusionList, globalOutboundIPRangeInclusionList)

	networkInterfaceExclusionList := wh.configurator.GetMeshConfig().Spec.Traffic.NetworkInterfaceExclusionList

	// Add the init container to the pod spec
	initContainer := getInitContainerSpec(constants.InitContainerName, wh.configurator, outboundIPRangeExclusionList, outboundIPRangeInclusionList, outboundPortExclusionList, inboundPortExclusionList, wh.configurator.IsPrivilegedInitContainer(), wh.osmContainerPullPolicy, networkInterfaceExclusionList)
	pod.Spec.InitContainers = append(pod.Spec.InitContainers, initContainer)

	return nil
}

func makePatches(req *admissionv1.AdmissionRequest, pod *corev1.Pod) []jsonpatch.JsonPatchOperation {
	original := req.Object.Raw
	current, err := json.Marshal(pod)
	if err != nil {
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrMarshallingKubernetesResource)).
			Msgf("Error marshaling Pod with UID=%s", pod.ObjectMeta.UID)
	}
	admissionResponse := admission.PatchResponseFromRaw(original, current)
	return admissionResponse.Patches
}

func getProxyUUID(pod *corev1.Pod) (string, bool) {
	// kubectl debug does not recreate the object with the same metadata
	for _, volume := range pod.Spec.Volumes {
		if volume.Name == envoyBootstrapConfigVolume {
			return strings.TrimPrefix(volume.Secret.SecretName, bootstrapSecretPrefix), true
		}
	}
	return "", false
}
