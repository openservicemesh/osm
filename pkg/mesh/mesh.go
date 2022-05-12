package mesh

import (
	"github.com/google/uuid"
	corev1 "k8s.io/api/core/v1"

	"github.com/openservicemesh/osm/pkg/constants"
)

// ProxyLabelExists returns a boolean indicating if the pod is part of a mesh.
// Note that this is NOT a definitive check indicating pod membership in a mesh
// as the label may continue to exist even if the pod is removed from the mesh.
func ProxyLabelExists(pod corev1.Pod) bool {
	// osm-controller adds a unique label to a pod when it is added to a mesh
	proxyUUID, proxyLabelSet := pod.Labels[constants.EnvoyUniqueIDLabelName]
	return proxyLabelSet && isValidUUID(proxyUUID)
}

func isValidUUID(u string) bool {
	_, err := uuid.Parse(u)
	return err == nil
}
