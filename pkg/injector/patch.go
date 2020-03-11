package injector

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/golang/glog"
	corev1 "k8s.io/api/core/v1"

	"github.com/deislabs/smc/pkg/certificate"
	"github.com/deislabs/smc/pkg/constants"
	"github.com/deislabs/smc/pkg/endpoint"
	"github.com/deislabs/smc/pkg/utils"
)

func (wh *Webhook) createPatch(pod *corev1.Pod, namespace string) ([]byte, error) {
	// Start patching the spec
	var patches []JSONPatchOperation
	glog.Infof("Patching POD spec: service-account=%s, namespace=%s", pod.Spec.ServiceAccountName, namespace)

	// Get the service for a service account
	namespacedSvcAcc := endpoint.NamespacedServiceAccount{
		Namespace:      namespace,
		ServiceAccount: pod.Spec.ServiceAccountName,
	}
	services := wh.meshCatalog.GetServicesByServiceAccountName(namespacedSvcAcc, true)

	if len(services) == 0 {
		// No services found for this service account, don't patch
		return nil, fmt.Errorf("No service found for service account %q", pod.Spec.ServiceAccountName)
	}

	// TODO(shashank): Don't assume 1-1 mapping between service account and service
	service := services[0]
	if !strings.Contains(service.String(), constants.NamespaceServiceDelimiter) {
		panic("Service name should be of the form: namespace/service")
	}

	serviceName := strings.Split(service.String(), constants.NamespaceServiceDelimiter)[1]

	// Issue a certificate for the envoy fronting the service
	cn := certificate.CommonName(utils.NewCertCommonNameWithUUID(fmt.Sprintf("%s.%s.smc.mesh", serviceName, namespace))) // TODO: Don't hardcode domain
	cert, err := wh.certManager.IssueCertificate(cn)
	if err != nil {
		glog.Errorf("Failed to issue TLS certificate for Envoy: %s", err)
		return nil, err
	}

	// Create kube secret for TLS cert and key used For Envoy to communicate with xDS
	envoyTLSSecretName := fmt.Sprintf("tls-%s", pod.Spec.ServiceAccountName)
	_, err = wh.createEnvoyTLSSecret(envoyTLSSecretName, namespace, cert.GetCertificateChain(), cert.GetPrivateKey())
	if err != nil {
		glog.Errorf("Failed to create TLS secret for Envoy sidecar: %s", err)
		return nil, err
	}

	// Create volume for envoy TLS secret
	patches = append(patches, addVolume(
		pod.Spec.Volumes,
		getVolumeSpec(envoyTLSSecretName),
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
	envoySidecarSpec, err := getEnvoySidecarContainerSpec(pod, &envoySidecarData)
	patches = append(patches, addContainer(
		pod.Spec.Containers,
		[]corev1.Container{envoySidecarSpec},
		"/spec/containers")...,
	)

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
