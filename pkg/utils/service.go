package utils

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"

	"github.com/openservicemesh/osm/pkg/service"
)

// K8sSvcToMeshSvc converts a Kubernetes service to a MeshService.
func K8sSvcToMeshSvc(svc *corev1.Service) service.MeshService {
	return service.MeshService{
		Namespace: svc.Namespace,
		Name:      svc.Name,
	}
}

// GetTrafficTargetName returns the name for a TrafficTarget with the given source and destination.
func GetTrafficTargetName(name string, srcSvc service.MeshService, destSvc service.MeshService) string {
	if name != "" {
		return fmt.Sprintf("%s:%s->%s", name, srcSvc, destSvc)
	}
	return fmt.Sprintf("%s->%s", srcSvc, destSvc)
}
