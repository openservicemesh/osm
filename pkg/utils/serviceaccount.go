package utils

import (
	corev1 "k8s.io/api/core/v1"

	"github.com/openservicemesh/osm/pkg/service"
)

// SvcAccountToK8sSvcAccount converts a Kubernetes service to a MeshService.
func SvcAccountToK8sSvcAccount(svcAccount *corev1.ServiceAccount) service.K8sServiceAccount {
	return service.K8sServiceAccount{
		Namespace: svcAccount.Namespace,
		Name:      svcAccount.Name,
	}
}
