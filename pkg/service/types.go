// Package service models an instance of a service managed by OSM controller and utility routines associated with it.
package service

import (
	"fmt"

	policyv1alpha1 "github.com/openservicemesh/osm/pkg/apis/policy/v1alpha1"
)

// Locality is the relative locality of a service. ie: if a service is being accessed from the same namespace or a
// remote cluster.
type Locality int

const (
	// LocalNS refers to the local namespace within the local cluster.
	LocalNS Locality = iota

	// LocalCluster refers to access within the cluster, but not within the same namespace.
	LocalCluster

	// RemoteCluster refers to access from a different cluster.
	RemoteCluster
)

// MeshService is the struct representing a service (Kubernetes or otherwise) within the service mesh.
type MeshService struct {
	// If the service resides on a Kubernetes service, this would be the Kubernetes namespace.
	Namespace string

	// The name of the service
	Name string

	// Subdomain is the subdomain of the service. This is only populated if the service is a headless service backed by a
	// single pod.
	Subdomain string

	// Port is the port number that clients use to access the service.
	// This can be different than MeshService.TargetPort which represents the actual port number
	// the application is accepting connections on.
	// Port maps to ServicePort.Port in k8s: https://pkg.go.dev/k8s.io/api/core/v1#ServicePort
	Port uint16

	// TargetPort is the port number on which an application accept traffic directed to this MeshService
	// This can be different than MeshService.Port in k8s.
	// TargetPort maps to ServicePort.TargetPort in k8s: https://pkg.go.dev/k8s.io/api/core/v1#ServicePort
	TargetPort uint16

	// Protocol is the protocol served by the service's port
	Protocol string
}

// NamespacedKey is the key (i.e. namespace + Name) with which to lookup the backing service within the provider
func (ms MeshService) NamespacedKey() string {
	return fmt.Sprintf("%s/%s", ms.Namespace, ms.Name)
}

// FullName is the name of the service including the subdomain.
func (ms MeshService) FullName() string {
	if ms.Subdomain != "" {
		return fmt.Sprintf("%s.%s", ms.Subdomain, ms.Name)
	}
	return ms.Name
}

// String returns the string representation of the given MeshService.
// SHOULD NOT BE USED AS A MAPPING FOR ANYTHING. Use NamespacedKey and Subdomain
func (ms MeshService) String() string {
	return fmt.Sprintf("%s/%s", ms.Namespace, ms.Name)
}

// EnvoyClusterName is the name of the cluster corresponding to the MeshService in Envoy
func (ms MeshService) EnvoyClusterName() string {
	return fmt.Sprintf("%s/%s|%d", ms.Namespace, ms.FullName(), ms.TargetPort)
}

// EnvoyLocalClusterName is the name of the local cluster corresponding to the MeshService in Envoy
func (ms MeshService) EnvoyLocalClusterName() string {
	return fmt.Sprintf("%s/%s|%d|local", ms.Namespace, ms.FullName(), ms.TargetPort)
}

// FQDN is similar to String(), but uses a dot separator and is in a different order.
func (ms MeshService) FQDN() string {
	return fmt.Sprintf("%s.%s.svc.cluster.local", ms.FullName(), ms.Namespace)
}

// OutboundTrafficMatchName returns the MeshService outbound traffic match name
func (ms MeshService) OutboundTrafficMatchName() string {
	return fmt.Sprintf("outbound_%s_%d_%s", ms, ms.Port, ms.Protocol)
}

// InboundTrafficMatchName returns the MeshService inbound traffic match name
func (ms MeshService) InboundTrafficMatchName() string {
	return fmt.Sprintf("inbound_%s_%d_%s", ms, ms.TargetPort, ms.Protocol)
}

// IngressTrafficMatchName returns the ingress traffic match name
func IngressTrafficMatchName(name, namespace string, targetPort uint16, protocol string) string {
	return fmt.Sprintf("ingress_%s/%s_%d_%s", namespace, name, targetPort, protocol)
}

// RateLimitServiceClusterName returns the cluster name used for the global rate limit service
func RateLimitServiceClusterName(svc policyv1alpha1.RateLimitServiceSpec) string {
	return fmt.Sprintf("%s|%d", svc.Host, svc.Port)
}

// ClusterName is a type for a service name
type ClusterName string

// String returns the given ClusterName type as a string
func (c ClusterName) String() string {
	return string(c)
}

// WeightedCluster is a struct of a cluster and is weight that is backing a service
type WeightedCluster struct {
	ClusterName ClusterName `json:"cluster_name:omitempty"`
	Weight      int         `json:"weight:omitempty"`
}
