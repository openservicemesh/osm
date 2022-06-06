// Package identity implements types and utility routines related to the identity of a workload, as used within OSM.
package identity

import (
	"fmt"
	"strings"
)

const (
	// namespaceNameSeparator used for marshalling/unmarshalling MeshService to a string or vice versa
	namespaceNameSeparator = "/"

	// ClusterLocalTrustDomain is the trust domain for the local kubernetes cluster
	ClusterLocalTrustDomain = "cluster.local"
)

// ServiceIdentity is the type used to represent the identity for a service
// For Kubernetes services this string will be in the format: <ServiceAccount>.<Namespace>.cluster.local
type ServiceIdentity string

// New returns a new ServiceIdentity for the given name and namespace.
func New(name, namespace string) ServiceIdentity {
	return ServiceIdentity(fmt.Sprintf("%s.%s.%s", name, namespace, ClusterLocalTrustDomain))
}

// WildcardServiceIdentity is a wildcard to match all service identities
const WildcardServiceIdentity ServiceIdentity = "*"

// String returns the ServiceIdentity as a string
func (si ServiceIdentity) String() string {
	return string(si)
}

// IsWildcard determines if the ServiceIdentity is a wildcard
func (si ServiceIdentity) IsWildcard() bool {
	return si == WildcardServiceIdentity
}

// ToK8sServiceAccount converts a ServiceIdentity to a K8sServiceAccount to help with transition from K8sServiceAccount to ServiceIdentity
func (si ServiceIdentity) ToK8sServiceAccount() K8sServiceAccount {
	// By convention as of release-v0.8 ServiceIdentity is in the format: <ServiceAccount>.<Namespace>.cluster.local
	// We can split by "." and will have service account in the first position and namespace in the second.
	chunks := strings.Split(si.String(), ".")
	name := chunks[0]
	namespace := chunks[1]
	return K8sServiceAccount{
		Namespace: namespace,
		Name:      name,
	}
}

// K8sServiceAccount is a type for a namespaced service account
type K8sServiceAccount struct {
	Namespace string
	Name      string
}

// String returns the string representation of the service account object
func (sa K8sServiceAccount) String() string {
	return fmt.Sprintf("%s%s%s", sa.Namespace, namespaceNameSeparator, sa.Name)
}

// ToServiceIdentity converts K8sServiceAccount to the newer ServiceIdentity
func (sa K8sServiceAccount) ToServiceIdentity() ServiceIdentity {
	return New(sa.Name, sa.Namespace)
}
