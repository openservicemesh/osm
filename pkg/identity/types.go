// Package identity implements types and utility routines related to the identity of a workload, as used within OSM.
package identity

// ServiceIdentity is the type used to represent the identity for a service
// For Kubernetes services this string will be in the format: <ServiceAccount>.<Namespace>.cluster.local
type ServiceIdentity string

// String returns the ServiceIdentity as a string
func (si ServiceIdentity) String() string {
	return string(si)
}
