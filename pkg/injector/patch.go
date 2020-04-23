package injector

import (
	"encoding/json"
	"fmt"
	"path"
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"

	"github.com/open-service-mesh/osm/pkg/constants"
	"github.com/open-service-mesh/osm/pkg/utils"
)

const (
	prometheusScrapeAnnotation = "prometheus.io/scrape"
	prometheusPortAnnotation   = "prometheus.io/port"
	prometheusPathAnnotation   = "prometheus.io/path"
)

func (wh *Webhook) createPatch(pod *corev1.Pod, namespace string) ([]byte, error) {
	// Start patching the spec
	var patches []JSONPatchOperation
	log.Info().Msgf("Patching POD spec: service-account=%s, namespace=%s", pod.Spec.ServiceAccountName, namespace)

	serviceName := getServiceName(pod)

	// Issue a certificate for the proxy sidecar
	subDomain := "osm.mesh" // TODO: don't hardcode this
	cn := utils.NewCertCommonNameWithUUID(serviceName, namespace, subDomain)
	bootstrapCertificate, err := wh.certManager.IssueCertificate(cn)
	if err != nil {
		log.Error().Err(err).Msgf("Error issuing bootstrap certificate for Envoy with CN=%s", cn)
		return nil, err
	}

	// Create kube secret for Envoy bootstrap config
	envoyBootstrapConfigName := fmt.Sprintf("envoy-bootstrap-config-%s", serviceName)
	_, err = wh.createEnvoyBootstrapConfig(envoyBootstrapConfigName, namespace, wh.osmNamespace, bootstrapCertificate)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create bootstrap config for Envoy sidecar")
		return nil, err
	}

	// Create volume for envoy TLS secret
	patches = append(patches, addVolume(
		pod.Spec.Volumes,
		getVolumeSpec(envoyBootstrapConfigName),
		"/spec/volumes")...,
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
		"/spec/initContainers")...,
	)

	// Add the Envoy sidecar
	envoySidecarData := EnvoySidecarData{
		Name:    envoySidecarContainerName,
		Image:   wh.config.SidecarImage,
		Service: serviceName,
	}
	patches = append(patches, addContainer(
		pod.Spec.Containers,
		[]corev1.Container{getEnvoySidecarContainerSpec(&envoySidecarData)},
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

	return json.Marshal(patches)
}

func getServiceName(pod *corev1.Pod) string {
	// Check if the POD is annotated for injection
	service, found := pod.ObjectMeta.Annotations[annotationService]
	if !found {
		log.Info().Msgf("Missing annotation '%s', using the ServiceAccount name for the Service", annotationService)
		return pod.Spec.ServiceAccountName
	}
	return service
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

// escapeJSONPointerValue escapes a JSON value as per https://tools.ietf.org/html/rfc6901.
// Character '~' is encoded to '~0' and '/' is encoded to '~1' because
// they have special meanings in JSON Pointer.
func escapeJSONPointerValue(s string) string {
	s = strings.Replace(s, "~", "~0", -1)
	s = strings.Replace(s, "/", "~1", -1)
	return s
}
