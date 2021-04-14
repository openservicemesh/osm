package identity

import (
	"strings"
)

const (
	// ClusterLocalTrustDomain is the trust domain for the local kubernetes cluster
	ClusterLocalTrustDomain = "cluster.local"

	identityDelimiter = "."
)

// GetKubernetesServiceIdentity returns the ServiceIdentity based on Kubernetes ServiceAccount and a trust domain
func GetKubernetesServiceIdentity(svcAccount K8sServiceAccount, trustDomain string) ServiceIdentity {
	si := strings.Join([]string{svcAccount.Name, svcAccount.Namespace, trustDomain}, identityDelimiter)
	return ServiceIdentity(si)
}
