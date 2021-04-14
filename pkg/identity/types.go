// Package identity implements types and utility routines related to the identity of a workload, as used within OSM.
package identity

import (
	"errors"
	"fmt"
	"strings"

	"github.com/openservicemesh/osm/pkg/certificate"
)

const (
	// namespaceNameSeparator used upon marshalling/unmarshalling MeshService to a string or vice versa
	namespaceNameSeparator = "/"
)

var (
	// ErrInvalidServiceAccountStringFormat is an error returned when the K8sServiceAccount string cannot be parsed (is invalid for some reason)
	ErrInvalidServiceAccountStringFormat = errors.New("invalid namespaced service string format")
)

// ServiceIdentity is the type used to represent the identity for a service
type ServiceIdentity struct {
	kind ServiceIdentityKind

	// serviceAccount is only used when the kind of ServiceIdentity is KubernetesServiceAccount
	serviceAccount K8sServiceAccount
	trustDomain    string
}

type ServiceIdentityKind string

const (
	// KubernetesServiceAccount is a ServiceIdentity derived from a Kubernetes Service Account on a specific cluster
	KubernetesServiceAccount ServiceIdentityKind = "kubernetes-service-account"
)

// String returns the ServiceIdentity as a string
// For Kubernetes services this string will be in the format: <ServiceAccount>.<Namespace>.cluster.local
func (si ServiceIdentity) String() string {
	if si.kind != KubernetesServiceAccount {
		panic(fmt.Sprintf("ServiceIdentity kind %q is not implemented", si.kind))
	}
	return fmt.Sprintf("%s.%s.%s", si.serviceAccount.Name, si.serviceAccount.Namespace, si.trustDomain)
}

// GetCertificateCommonName returns a certificate CommonName compliant with RFC-1123 (https://tools.ietf.org/html/rfc1123) DNS name.
func (si ServiceIdentity) GetCertificateCommonName() certificate.CommonName {
	return certificate.CommonName(si.String())
}

// Deprecated:ToK8sServiceAccount converts a ServiceIdentity to a K8sServiceAccount to help with transition from K8sServiceAccount to ServiceIdentity
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

// Deprecated:K8sServiceAccount is a type for a namespaced service account
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
func (sa K8sServiceAccount) ToServiceIdentity() ServiceIdentity {
	return ServiceIdentity{
		kind: KubernetesServiceAccount,
		serviceAccount: K8sServiceAccount{
			Name:      sa.Name,
			Namespace: sa.Namespace,
		},
		trustDomain: ClusterLocalTrustDomain,
	}
}

// UnmarshalK8sServiceAccount unmarshals a K8sServiceAccount type from a string
func UnmarshalK8sServiceAccount(str string) (*K8sServiceAccount, error) {
	slices := strings.Split(str, namespaceNameSeparator)
	if len(slices) != 2 {
		return nil, ErrInvalidServiceAccountStringFormat
	}

	// Make sure the slices are not empty. Split might actually leave empty slices.
	for _, sep := range slices {
		if len(sep) == 0 {
			return nil, ErrInvalidServiceAccountStringFormat
		}
	}

	return &K8sServiceAccount{
		Namespace: slices[0],
		Name:      slices[1],
	}, nil
}
