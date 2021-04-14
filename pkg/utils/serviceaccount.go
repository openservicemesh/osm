package utils

import (
	"github.com/openservicemesh/osm/pkg/identity"
	corev1 "k8s.io/api/core/v1"
)

// SvcAccountToK8sSvcAccount converts a Kubernetes service to a MeshService.
func SvcAccountToK8sSvcAccount(svcAccount *corev1.ServiceAccount) identity.K8sServiceAccount {
	return identity.K8sServiceAccount{
		Namespace: svcAccount.Namespace,
		Name:      svcAccount.Name,
	}
}
