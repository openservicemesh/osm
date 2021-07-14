package identity

const (
	// ClusterLocalTrustDomain is the trust domain for the local kubernetes cluster
	ClusterLocalTrustDomain = "cluster.local"

	identityDelimiter = "."
)

// GetKubernetesServiceIdentity returns the ServiceIdentity based on Kubernetes ServiceAccount and a trust domain
func GetKubernetesServiceIdentity(svcAccount K8sServiceAccount, trustDomain string) ServiceIdentity {
	return ServiceIdentity{svcAccount.Name, svcAccount.Namespace, trustDomain}
}
