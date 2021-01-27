package injector

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"gomodules.xyz/jsonpatch/v2"
	"k8s.io/api/admission/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/metricsstore"
)

func (wh *mutatingWebhook) createPatch(pod *corev1.Pod, req *v1beta1.AdmissionRequest, proxyUUID uuid.UUID) ([]byte, error) {
	namespace := req.Namespace

	// Issue a certificate for the proxy sidecar - used for Envoy to connect to XDS (not Envoy-to-Envoy connections)
	cn := catalog.NewCertCommonNameWithProxyID(proxyUUID, pod.Spec.ServiceAccountName, namespace)
	log.Debug().Msgf("Patching POD spec: service-account=%s, namespace=%s with certificate CN=%s", pod.Spec.ServiceAccountName, namespace, cn)
	startTime := time.Now()
	bootstrapCertificate, err := wh.certManager.IssueCertificate(cn, constants.XDSCertificateValidityPeriod)
	if err != nil {
		log.Error().Err(err).Msgf("Error issuing bootstrap certificate for Envoy with CN=%s", cn)
		return nil, err
	}
	elapsed := time.Since(startTime)

	metricsstore.DefaultMetricsStore.CertXdsIssuedCount.Inc()
	metricsstore.DefaultMetricsStore.CertXdsIssuedTime.
		WithLabelValues(cn.String()).Observe(elapsed.Seconds())
	originalHealthProbes := rewriteHealthProbes(pod)

	wh.meshCatalog.ExpectProxy(cn)
	// Create the bootstrap configuration for the Envoy proxy for the given pod
	envoyBootstrapConfigName := fmt.Sprintf("envoy-bootstrap-config-%s", proxyUUID)
	if _, err = wh.createEnvoyBootstrapConfig(envoyBootstrapConfigName, namespace, wh.osmNamespace, bootstrapCertificate, originalHealthProbes); err != nil {
		log.Error().Err(err).Msg("Failed to create bootstrap config for Envoy sidecar")
		return nil, err
	}

	// Create volume for envoy TLS secret
	pod.Spec.Volumes = append(pod.Spec.Volumes, getVolumeSpec(envoyBootstrapConfigName)...)

	// Add the Init Container
	initContainer := getInitContainerSpec(constants.InitContainerName, wh.config.InitContainerImage, wh.configurator.GetOutboundIPRangeExclusionList())
	pod.Spec.InitContainers = append(pod.Spec.InitContainers, initContainer)

	// envoyNodeID and envoyClusterID are required for Envoy proxy to start.
	envoyNodeID := pod.Spec.ServiceAccountName

	// envoyCluster ID will be used as an identifier to the tracing sink
	envoyClusterID := fmt.Sprintf("%s.%s", pod.Spec.ServiceAccountName, namespace)

	// Add the Envoy sidecar
	sidecar := getEnvoySidecarContainerSpec(constants.EnvoyContainerName, wh.config.SidecarImage, envoyNodeID, envoyClusterID, wh.configurator, originalHealthProbes)
	pod.Spec.Containers = append(pod.Spec.Containers, sidecar)

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

	return json.Marshal(makePatches(req, pod))
}

func makePatches(req *v1beta1.AdmissionRequest, pod *corev1.Pod) []jsonpatch.JsonPatchOperation {
	original := req.Object.Raw
	current, err := json.Marshal(pod)
	if err != nil {
		log.Error().Err(err).Msgf("Error marshaling Pod with UID=%s", pod.ObjectMeta.UID)
	}
	admissionResponse := admission.PatchResponseFromRaw(original, current)
	return admissionResponse.Patches
}
