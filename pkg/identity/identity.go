package identity

import (
	"fmt"
	"strings"
)

const fqdnSeparator = "."

// NewFromServiceAccountString is a helper to convert "foo.bar.cluster.local" into ServiceIdentity
func NewFromServiceAccountString(fqdn string) ServiceIdentity {
	svcIdent, err := NewFromRFC1123(fqdn, KubernetesServiceAccount)
	if err != nil {
		log.Err(err).Msgf("Error converting %s to ServiceIdentity", fqdn)
		return ServiceIdentity{}
	}
	return svcIdent
}

func NewFromRFC1123(fqdn string, kind ServiceIdentityKind) (ServiceIdentity, error) {
	if kind != KubernetesServiceAccount {
		panic(fmt.Sprintf("ServiceIdentity kind %q is not implemented", kind))
	}

	// By convention as of release-v0.8 ServiceIdentity is in the format: <ServiceAccount>.<Namespace>.cluster.local
	// We can split by "." and will have service account in the first position and namespace in the second.
	chunks := strings.Split(fqdn, fqdnSeparator)
	name := chunks[0]
	namespace := chunks[1]

	return ServiceIdentity{
		kind: KubernetesServiceAccount,
		serviceAccount: K8sServiceAccount{
			Namespace: namespace,
			Name:      name,
		},
		trustDomain: strings.Join(chunks[2:], fqdnSeparator),
	}, nil
}
