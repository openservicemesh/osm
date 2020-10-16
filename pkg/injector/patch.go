package injector

import (
	"encoding/json"
	"fmt"
	"path"
	"sort"
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"

	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/constants"
)

const (
	// Patch Operations
	addOperation     = "add"
	replaceOperation = "replace"

	volumesBasePath        = "/spec/volumes"
	initContainersBasePath = "/spec/initContainers"
	labelsPath             = "/metadata/labels"
)

func (wh *webhook) createPatch(pod *corev1.Pod, namespace, proxyUUID string) ([]byte, error) {
	// Start patching the spec
	var patches []JSONPatchOperation

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
	patches = append(patches, addVolume(
		pod.Spec.Volumes,
		getVolumeSpec(envoyBootstrapConfigName),
		volumesBasePath)...,
	)

	// Add the Init Container
	initContainerData := InitContainer{
		Name:  constants.InitContainerName,
		Image: wh.config.InitContainerImage,
	}
	initContainerSpec, err := getInitContainerSpec(&initContainerData)
	if err != nil {
		return nil, err
	}
	patches = append(patches, addContainer(
		pod.Spec.InitContainers,
		[]corev1.Container{initContainerSpec},
		initContainersBasePath)...,
	)

	// envoyNodeID and envoyClusterID are required for Envoy proxy to start.
	envoyNodeID := pod.Spec.ServiceAccountName

	// envoyCluster ID will be used as an identifier to the tracing sink
	envoyClusterID := fmt.Sprintf("%s.%s", pod.Spec.ServiceAccountName, namespace)

	patches = append(patches, addContainer(
		pod.Spec.Containers,
		getEnvoySidecarContainerSpec(constants.EnvoyContainerName, wh.config.SidecarImage, envoyNodeID, envoyClusterID, wh.configurator),
		"/spec/containers")...,
	)

	enableMetrics, err := wh.isMetricsEnabled(namespace)
	if err != nil {
		log.Error().Err(err).Msgf("Error checking if namespace %s is enabled for metrics", namespace)
		return nil, err
	}
	if enableMetrics {
		// Patch annotations
		prometheusAnnotations := map[string]string{
			constants.PrometheusScrapeAnnotation: strconv.FormatBool(true),
			constants.PrometheusPortAnnotation:   strconv.Itoa(constants.EnvoyPrometheusInboundListenerPort),
			constants.PrometheusPathAnnotation:   constants.PrometheusScrapePath,
		}
		patches = append(patches, updateAnnotation(
			pod.Annotations,
			prometheusAnnotations,
			"/metadata/annotations")...,
		)
	}

	patches = append(patches, *updateLabels(pod, proxyUUID))

	return json.Marshal(patches)
}

func addVolume(target, add []corev1.Volume, basePath string) (patch []JSONPatchOperation) {
	isFirst := len(target) == 0 // target is empty, use this to create the first item
	var value interface{}
	for _, volume := range add {
		value = volume
		volumePath := basePath
		if isFirst {
			isFirst = false
			value = []corev1.Volume{volume}
		} else {
			volumePath += "/-"
		}
		patch = append(patch, JSONPatchOperation{
			Op:    addOperation,
			Path:  volumePath,
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
			Op:    addOperation,
			Path:  path,
			Value: value,
		})
	}
	return patch
}

// createMapPatchOperation is used specifically for the very first 'add' operation
// for a non-existent 'path'.
func createMapPatchOperation(key, path, value string) JSONPatchOperation {
	return JSONPatchOperation{
		Op:   addOperation,
		Path: path,
		Value: map[string]string{
			key: value,
		},
	}
}

// createPatch is used for add and replace operations on existing paths.
func createPatch(key, basePath, patchOp, value string) JSONPatchOperation {
	return JSONPatchOperation{
		Op:    patchOp,
		Path:  path.Join(basePath, escapeJSONPointerValue(key)),
		Value: value,
	}
}

// getSortedKeys returns the keys of a map in sorted string slice.
// The purpose of this is to provide determinism when iterating over a map.
func getSortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func updateAnnotation(target, add map[string]string, basePath string) []JSONPatchOperation {
	var patches []JSONPatchOperation

	// If the target does not exist we need to create it
	noTarget := len(target) == 0

	// We iterate over the sorted keys in order to have a deterministic output from this function.
	// One of many areas where determinism is useful and worth the cost is testing this function.
	for _, key := range getSortedKeys(add) {
		value := add[key]
		if noTarget {
			patches = append(patches, createMapPatchOperation(key, basePath, value))
			noTarget = false
			continue
		}

		patchOp := addOperation
		if target[key] != "" {
			patchOp = replaceOperation
		}

		patches = append(patches, createPatch(key, basePath, patchOp, value))
	}

	return patches
}

// This function will append a label to the pod, which points to the unique Envoy ID used in the
// xDS certificate for that Envoy. This label will help xDS match the actual pod to the Envoy that
// connects to xDS (with the certificate's CN matching this label).
func updateLabels(pod *corev1.Pod, envoyUID string) *JSONPatchOperation {
	if len(pod.Labels) == 0 {
		return &JSONPatchOperation{
			Op:    addOperation,
			Path:  labelsPath,
			Value: map[string]string{constants.EnvoyUniqueIDLabelName: envoyUID},
		}
	}

	getOp := func() string {
		if _, exists := pod.Labels[constants.EnvoyUniqueIDLabelName]; exists {
			return replaceOperation
		}
		return addOperation
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
