// Package identity implements types and utility routines related to the identity of a workload, as used within OSM.
package identity

import (
	"errors"
	"fmt"
	"strings"
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
// For Kubernetes services this string will be in the format: <ServiceAccount>.<Namespace>.cluster.local
type ServiceIdentity string

// String returns the ServiceIdentity as a string
func (si ServiceIdentity) String() string {
	return string(si)
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
