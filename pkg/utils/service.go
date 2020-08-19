package utils

import (
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
