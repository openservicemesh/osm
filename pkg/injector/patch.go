package injector

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/google/uuid"
	"gomodules.xyz/jsonpatch/v2"
	"k8s.io/api/admission/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/constants"
)

func (wh *webhook) createPatch(pod *corev1.Pod, req *v1beta1.AdmissionRequest, proxyUUID uuid.UUID) ([]byte, error) {
	namespace := req.Namespace

	// Issue a certificate for the proxy sidecar - used for Envoy to connect to XDS (not Envoy-to-Envoy connections)
	cn := catalog.NewCertCommonNameWithProxyID(proxyUUID, pod.Spec.ServiceAccountName, namespace)
	log.Info().Msgf("Patching POD spec: service-account=%s, namespace=%s with certificate CN=%s", pod.Spec.ServiceAccountName, namespace, cn)
	bootstrapCertificate, err := wh.certManager.IssueCertificate(cn, constants.XDSCertificateValidityPeriod)
	if err != nil {
		log.Error().Err(err).Msgf("Error issuing bootstrap certificate for Envoy with CN=%s", cn)
		return nil, err
	}

	wh.meshCatalog.ExpectProxy(cn)

	// Create kube secret for Envoy bootstrap config
	envoyBootstrapConfigName := fmt.Sprintf("envoy-bootstrap-config-%s", proxyUUID)
	_, err = wh.createEnvoyBootstrapConfig(envoyBootstrapConfigName, namespace, wh.osmNamespace, bootstrapCertificate)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create bootstrap config for Envoy sidecar")
		return nil, err
	}

	// Create volume for envoy TLS secret
	pod.Spec.Volumes = getVolumeSpec(envoyBootstrapConfigName)

	// Add the Init Container
	initContainerData := InitContainer{
		Name:  constants.InitContainerName,
		Image: wh.config.InitContainerImage,
	}
	initContainerSpec, err := getInitContainerSpec(&initContainerData)
	if err != nil {
		return nil, err
	}

	pod.Spec.InitContainers = []corev1.Container{initContainerSpec}

	// envoyNodeID and envoyClusterID are required for Envoy proxy to start.
	envoyNodeID := pod.Spec.ServiceAccountName

	// envoyCluster ID will be used as an identifier to the tracing sink
	envoyClusterID := fmt.Sprintf("%s.%s", pod.Spec.ServiceAccountName, namespace)

	sidecar := getEnvoySidecarContainerSpec(constants.EnvoyContainerName, wh.config.SidecarImage, envoyNodeID, envoyClusterID, wh.configurator)
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
	pod.Labels[constants.EnvoyUniqueIDLabelName] = proxyUUID.String()

	return json.Marshal(makePatches(req, pod))
}

func makePatches(req *v1beta1.AdmissionRequest, pod *corev1.Pod) []jsonpatch.JsonPatchOperation {
	original := req.Object.Raw
	current, err := json.Marshal(pod)
	if err != nil {
		log.Err(err).Msg("") // TODO
	}
	admissionResponse := admission.PatchResponseFromRaw(original, current)
	return admissionResponse.Patches
}
