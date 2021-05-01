package kubernetes

import (
	"strings"

	"github.com/google/uuid"
	v1 "k8s.io/api/core/v1"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/constants"
)

// GetPodFromCertificate returns the Kubernetes Pod object for a given certificate.
func (c Client) GetPodFromCertificate(cn certificate.CommonName) (*v1.Pod, error) {
	cnMeta, err := getCertificateCommonNameMeta(cn)
	if err != nil {
		return nil, err
	}

	log.Trace().Msgf("Looking for pod with label %q=%q", constants.EnvoyUniqueIDLabelName, cnMeta.ProxyUUID)
	podList := c.ListPods()
	var pods []v1.Pod
	for _, pod := range podList {
		if pod.Namespace != cnMeta.Namespace {
			continue
		}
		if envoyUUID, labelFound := pod.Labels[constants.EnvoyUniqueIDLabelName]; labelFound && envoyUUID == cnMeta.ProxyUUID.String() {
			pods = append(pods, *pod)
		}
	}

	if len(pods) == 0 {
		log.Error().Msgf("Did not find Pod with label %s = %s in namespace %s",
			constants.EnvoyUniqueIDLabelName, cnMeta.ProxyUUID, cnMeta.Namespace)
		return nil, ErrDidNotFindPodForCertificate
	}

	// --- CONVENTION ---
	// By Open Service Mesh convention the number of services a pod can belong to is 1
	// This is a limitation we set in place in order to make the mesh easy to understand and reason about.
	// When a pod belongs to more than one service XDS will not program the Envoy proxy, leaving it out of the mesh.
	if len(pods) > 1 {
		log.Error().Msgf("Found more than one pod with label %s = %s in namespace %s. There can be only one!",
			constants.EnvoyUniqueIDLabelName, cnMeta.ProxyUUID, cnMeta.Namespace)
		return nil, ErrMoreThanOnePodForCertificate
	}

	pod := pods[0]
	log.Trace().Msgf("Found Pod with UID=%s for proxyID %s", pod.ObjectMeta.UID, cnMeta.ProxyUUID)

	// Ensure the Namespace encoded in the certificate matches that of the Pod
	if pod.Namespace != cnMeta.Namespace {
		log.Warn().Msgf("Pod with UID=%s belongs to Namespace %s. The pod's xDS certificate was issued for Namespace %s",
			pod.ObjectMeta.UID, pod.Namespace, cnMeta.Namespace)
		return nil, ErrNamespaceDoesNotMatchCertificate
	}

	// Ensure the Name encoded in the certificate matches that of the Pod
	// TODO(draychev): check that the Kind matches too! [https://github.com/openservicemesh/osm/issues/3173]
	if pod.Spec.ServiceAccountName != cnMeta.ServiceAccount {
		// Since we search for the pod in the namespace we obtain from the certificate -- these namespaces will always match.
		log.Warn().Msgf("Pod with UID=%s belongs to ServiceAccount=%s. The pod's xDS certificate was issued for ServiceAccount=%s",
			pod.ObjectMeta.UID, pod.Spec.ServiceAccountName, cnMeta.ServiceAccount)
		return nil, ErrServiceAccountDoesNotMatchCertificate
	}

	return &pod, nil
}

func getCertificateCommonNameMeta(cn certificate.CommonName) (*certificateCommonNameMeta, error) {
	chunks := strings.Split(cn.String(), constants.DomainDelimiter)
	if len(chunks) < 3 {
		return nil, ErrInvalidCertificateCN
	}
	proxyUUID, err := uuid.Parse(chunks[0])
	if err != nil {
		log.Error().Err(err).Msgf("Error parsing %s into uuid.UUID", chunks[0])
		return nil, err
	}

	return &certificateCommonNameMeta{
		ProxyUUID: proxyUUID,
		// TODO(draychev): Use ServiceIdentity vs ServiceAccount
		ServiceAccount: chunks[1],
		Namespace:      chunks[2],
	}, nil
}

// certificateCommonNameMeta is the type that stores the metadata present in the CommonName field in a proxy's certificate
type certificateCommonNameMeta struct {
	ProxyUUID uuid.UUID
	// TODO(draychev): Change this to ServiceIdentity type (instead of string)
	ServiceAccount string
	Namespace      string
}
