package injector

import (
	"encoding/json"
	"fmt"
	"path"
	"strconv"
	"strings"

	"github.com/google/uuid"
	corev1 "k8s.io/api/core/v1"

	"github.com/open-service-mesh/osm/pkg/catalog"
	"github.com/open-service-mesh/osm/pkg/constants"
)

const (
	prometheusScrapeAnnotation = "prometheus.io/scrape"
	prometheusPortAnnotation   = "prometheus.io/port"
	prometheusPathAnnotation   = "prometheus.io/path"

	volumesBasePath        = "/spec/volumes"
	initContainersBasePath = "/spec/initContainers"
	labelsPath             = "/metadata/labels"
)

func (wh *webhook) createPatch(pod *corev1.Pod, namespace string) ([]byte, error) {
	// This string uniquely identifies the pod. Ideally this would be the pod.UID, but this is not available at this point.
	proxyUUID := uuid.New().String()

	// Start patching the spec
	var patches []JSONPatchOperation

	// Issue a certificate for the proxy sidecar - used for Envoy to connect to XDS (not Envoy-to-Envoy connections)
	cn := catalog.NewCertCommonNameWithProxyID(proxyUUID, pod.Spec.ServiceAccountName, namespace)
	log.Info().Msgf("Patching POD spec: service-account=%s, namespace=%s with certificate CN=%s", pod.Spec.ServiceAccountName, namespace, cn)
	validityPeriod := constants.XDSCertificateValidityPeriod
	bootstrapCertificate, err := wh.certManager.IssueCertificate(cn, &validityPeriod)
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
	patches = append(patches, addVolume(
		pod.Spec.Volumes,
		getVolumeSpec(envoyBootstrapConfigName),
		volumesBasePath)...,
	)

	// Add the Init Container
	initContainerData := InitContainerData{
		Name:  InitContainerName,
		Image: wh.config.InitContainerImage,
	}
	initContainerSpec, err := getInitContainerSpec(pod, &initContainerData)
	if err != nil {
		return nil, err
	}
	patches = append(patches, addContainer(
		pod.Spec.InitContainers,
		[]corev1.Container{initContainerSpec},
		initContainersBasePath)...,
	)

	patches = append(patches, addContainer(
		pod.Spec.Containers,
		getEnvoySidecarContainerSpec(envoyContainerName, wh.config.SidecarImage, proxyUUID, proxyUUID),
		"/spec/containers")...,
	)

	// Patch annotations
	prometheusAnnotations := map[string]string{
		prometheusScrapeAnnotation: strconv.FormatBool(true),
		prometheusPortAnnotation:   strconv.Itoa(constants.EnvoyPrometheusInboundListenerPort),
		prometheusPathAnnotation:   constants.PrometheusScrapePath,
	}
	patches = append(patches, updateAnnotation(
		pod.Annotations,
		prometheusAnnotations,
		"/metadata/annotations")...,
	)

	patches = append(patches, *updateLabels(pod, proxyUUID))

	return json.Marshal(patches)
}

func addVolume(target, add []corev1.Volume, basePath string) (patch []JSONPatchOperation) {
	isFirst := len(target) == 0 // target is empty, use this to create the first item
	var value interface{}
	for _, volume := range add {
		value = volume
		path := basePath
		if isFirst {
			isFirst = false
			value = []corev1.Volume{volume}
		} else {
			path += "/-"
		}
		patch = append(patch, JSONPatchOperation{
			Op:    "add",
			Path:  path,
			Value: value,
		})
	}
	return patch
}

func addContainer(target, add []corev1.Container, basePath string) (patch []JSONPatchOperation) {
	isFirst := len(target) == 0 // target is empty, use this to create the first item
	var value interface{}
	for _, container := range add {
		value = container
		path := basePath
		if isFirst {
			isFirst = false
			value = []corev1.Container{container}
		} else {
			path += "/-"
		}
		patch = append(patch, JSONPatchOperation{
			Op:    "add",
			Path:  path,
			Value: value,
		})
	}
	return patch
}

func updateAnnotation(target, add map[string]string, basePath string) (patch []JSONPatchOperation) {
	for key, value := range add {
		if target == nil {
			// First one will be a Create
			target = map[string]string{}
			patch = append(patch, JSONPatchOperation{
				Op:   "add",
				Path: basePath,
				Value: map[string]string{
					key: value,
				},
			})
		} else {
			// Update
			op := "add"
			if target[key] != "" {
				op = "replace"
			}
			patch = append(patch, JSONPatchOperation{
				Op:    op,
				Path:  path.Join(basePath, escapeJSONPointerValue(key)),
				Value: value,
			})
		}
	}

	return patch
}

// This function will append a label to the pod, which points to the unique Envoy ID used in the
// xDS certificate for that Envoy. This label will help xDS match the actual pod to the Envoy that
// connects to xDS (with the certificate's CN matching this label).
func updateLabels(pod *corev1.Pod, envoyUID string) *JSONPatchOperation {
	if len(pod.Labels) == 0 {
		return &JSONPatchOperation{
			Op:    "add",
			Path:  labelsPath,
			Value: map[string]string{constants.EnvoyUniqueIDLabelName: envoyUID},
		}
	}

	getOp := func() string {
		if _, exists := pod.Labels[constants.EnvoyUniqueIDLabelName]; exists {
			return "replace"
		}
		return "add"
	}

	return &JSONPatchOperation{
		Op:    getOp(),
		Path:  path.Join(labelsPath, constants.EnvoyUniqueIDLabelName),
		Value: envoyUID,
	}
}

// escapeJSONPointerValue escapes a JSON value as per https://tools.ietf.org/html/rfc6901.
// Character '~' is encoded to '~0' and '/' is encoded to '~1' because
// they have special meanings in JSON Pointer.
func escapeJSONPointerValue(s string) string {
	s = strings.Replace(s, "~", "~0", -1)
	s = strings.Replace(s, "/", "~1", -1)
	return s
}
