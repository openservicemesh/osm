// Package identity implements types and utility routines related to the identity of a workload, as used within OSM.
package identity

import (
	"fmt"
	"strings"
)

const (
	// namespaceNameSeparator used for marshalling/unmarshalling MeshService to a string or vice versa
	namespaceNameSeparator = "/"
)

// ServiceIdentity is the type used to represent the identity for a service
// For Kubernetes services this string will be in the format: <ServiceAccount>.<Namespace>.cluster.local
type ServiceIdentity struct {
	ServiceAccount string
	Namespace      string
	ClusterDomain  string
}

// NewServiceIdentityFromString creates a ServiceIdentity from a given string.
// The string will be splitted using "." into at most 3 chunks. The first two chunks do not contain ".". They will be the ServiceAccount and Namespace of the identity respectively. The chunk after the second "." will be the ClusterDomain. If any required chunk is not found, the corresponding field of the identity will be an empty string.
func NewServiceIdentityFromString(identityStr string) ServiceIdentity {
	id := ServiceIdentity{}
	chunks := strings.SplitN(identityStr, identityDelimiter, 3)

	if len(chunks) > 0 {
		id.ServiceAccount = chunks[0]
	}
	if len(chunks) > 1 {
		id.Namespace = chunks[1]
	}
	if len(chunks) > 2 {
		id.ClusterDomain = chunks[2]
	}

	return id
}

// String returns the ServiceIdentity as a string
func (si ServiceIdentity) String() string {
	result := si.ServiceAccount
	if si.Namespace == "" {
		return result
	}
	result += fmt.Sprintf(".%s", si.Namespace)

	if si.ClusterDomain == "" {
		return result
	}
	result += fmt.Sprintf(".%s", si.ClusterDomain)

	return result
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

// IsEmpty returns true if the given service account object is empty
func (sa K8sServiceAccount) IsEmpty() bool {
	return (K8sServiceAccount{}) == sa
}

// ToServiceIdentity converts K8sServiceAccount to the newer ServiceIdentity
// TODO(draychev): ToServiceIdentity is used in many places to ease with transition from K8sServiceAccount to ServiceIdentity and should be removed (not everywhere) - [https://github.com/openservicemesh/osm/issues/2218]
func (sa K8sServiceAccount) ToServiceIdentity() ServiceIdentity {
	return ServiceIdentity{sa.Name, sa.Namespace, ClusterLocalTrustDomain}
}
