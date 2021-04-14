package identity

const (
	// ClusterLocalTrustDomain is the trust domain for the local kubernetes cluster
	ClusterLocalTrustDomain = "cluster.local"
)

// NewFromKubernetesServiceAccount returns the ServiceIdentity based on Kubernetes ServiceAccount and a trust domain
func NewFromKubernetesServiceAccount(svcAccount K8sServiceAccount, trustDomain string) ServiceIdentity {
	return ServiceIdentity{
		kind: KubernetesServiceAccount,
		serviceAccount: K8sServiceAccount{
			Namespace: svcAccount.Namespace,
			Name:      svcAccount.Name,
		},
		trustDomain: trustDomain,
	}
}

func (si ServiceIdentity) GetKubernetesServiceAccount() (K8sServiceAccount, error) {
	if si.kind != KubernetesServiceAccount {
		return K8sServiceAccount{}, ErrNotAKubernetesServiceAccount
	}
	return si.serviceAccount, nil
}
