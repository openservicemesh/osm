// Package announcements provides the types and constants required to contextualize events received from the
// Kubernetes API server that are propagated internally within the control plane to trigger configuration changes.
package announcements

// Kind is used to record the kind of announcement
type Kind string

func (at Kind) String() string {
	return string(at)
}

const (
	// ProxyUpdate is the event kind used to trigger an update to subscribed proxies
	ProxyUpdate Kind = "proxy-update"

	// PodAdded is the type of announcement emitted when we observe an addition of a Kubernetes Pod
	PodAdded Kind = "pod-added"

	// PodDeleted the type of announcement emitted when we observe the deletion of a Kubernetes Pod
	PodDeleted Kind = "pod-deleted"

	// PodUpdated is the type of announcement emitted when we observe an update to a Kubernetes Pod
	PodUpdated Kind = "pod-updated"

	// ---

	// EndpointAdded is the type of announcement emitted when we observe an addition of a Kubernetes Endpoint
	EndpointAdded Kind = "endpoint-added"

	// EndpointDeleted the type of announcement emitted when we observe the deletion of a Kubernetes Endpoint
	EndpointDeleted Kind = "endpoint-deleted"

	// EndpointUpdated is the type of announcement emitted when we observe an update to a Kubernetes Endpoint
	EndpointUpdated Kind = "endpoint-updated"

	// ---

	// NamespaceAdded is the type of announcement emitted when we observe an addition of a Kubernetes Namespace
	NamespaceAdded Kind = "namespace-added"

	// NamespaceDeleted the type of announcement emitted when we observe the deletion of a Kubernetes Namespace
	NamespaceDeleted Kind = "namespace-deleted"

	// NamespaceUpdated is the type of announcement emitted when we observe an update to a Kubernetes Namespace
	NamespaceUpdated Kind = "namespace-updated"

	// ---

	// ServiceAdded is the type of announcement emitted when we observe an addition of a Kubernetes Service
	ServiceAdded Kind = "service-added"

	// ServiceDeleted the type of announcement emitted when we observe the deletion of a Kubernetes Service
	ServiceDeleted Kind = "service-deleted"

	// ServiceUpdated is the type of announcement emitted when we observe an update to a Kubernetes Service
	ServiceUpdated Kind = "service-updated"

	// ---

	// ServiceAccountAdded is the type of announcement emitted when we observe an addition of a Kubernetes Service Account
	ServiceAccountAdded Kind = "serviceaccount-added"

	// ServiceAccountDeleted the type of announcement emitted when we observe the deletion of a Kubernetes Service Account
	ServiceAccountDeleted Kind = "serviceaccount-deleted"

	// ServiceAccountUpdated is the type of announcement emitted when we observe an update to a Kubernetes Service
	ServiceAccountUpdated Kind = "serviceaccount-updated"

	// ---

	// TrafficSplitAdded is the type of announcement emitted when we observe an addition of a Kubernetes TrafficSplit
	TrafficSplitAdded Kind = "trafficsplit-added"

	// TrafficSplitDeleted the type of announcement emitted when we observe the deletion of a Kubernetes TrafficSplit
	TrafficSplitDeleted Kind = "trafficsplit-deleted"

	// TrafficSplitUpdated is the type of announcement emitted when we observe an update to a Kubernetes TrafficSplit
	TrafficSplitUpdated Kind = "trafficsplit-updated"

	// ---

	// RouteGroupAdded is the type of announcement emitted when we observe an addition of a Kubernetes RouteGroup
	RouteGroupAdded Kind = "routegroup-added"

	// RouteGroupDeleted the type of announcement emitted when we observe the deletion of a Kubernetes RouteGroup
	RouteGroupDeleted Kind = "routegroup-deleted"

	// RouteGroupUpdated is the type of announcement emitted when we observe an update to a Kubernetes RouteGroup
	RouteGroupUpdated Kind = "routegroup-updated"

	// ---

	// TCPRouteAdded is the type of announcement emitted when we observe an addition of a Kubernetes TCPRoute
	TCPRouteAdded Kind = "tcproute-added"

	// TCPRouteDeleted the type of announcement emitted when we observe the deletion of a Kubernetes TCPRoute
	TCPRouteDeleted Kind = "tcproute-deleted"

	// TCPRouteUpdated is the type of announcement emitted when we observe an update to a Kubernetes TCPRoute
	TCPRouteUpdated Kind = "tcproute-updated"

	// ---

	// TrafficTargetAdded is the type of announcement emitted when we observe an addition of a Kubernetes TrafficTarget
	TrafficTargetAdded Kind = "traffictarget-added"

	// TrafficTargetDeleted the type of announcement emitted when we observe the deletion of a Kubernetes TrafficTarget
	TrafficTargetDeleted Kind = "traffictarget-deleted"

	// TrafficTargetUpdated is the type of announcement emitted when we observe an update to a Kubernetes TrafficTarget
	TrafficTargetUpdated Kind = "traffictarget-updated"

	// ---

	// IngressAdded is the type of announcement emitted when we observe an addition of a Kubernetes Ingress
	IngressAdded Kind = "ingress-added"

	// IngressDeleted the type of announcement emitted when we observe the deletion of a Kubernetes Ingress
	IngressDeleted Kind = "ingress-deleted"

	// IngressUpdated is the type of announcement emitted when we observe an update to a Kubernetes Ingress
	IngressUpdated Kind = "ingress-updated"

	// ---

	// CertificateRotated is the type of announcement emitted when a certificate is rotated by the certificate provider
	CertificateRotated Kind = "certificate-rotated"

	// --- config.openservicemesh.io API events

	// MeshConfigAdded is the type of announcement emitted when we observe an addition of a Kubernetes MeshConfig
	MeshConfigAdded Kind = "meshconfig-added"

	// MeshConfigDeleted the type of announcement emitted when we observe the deletion of a Kubernetes MeshConfig
	MeshConfigDeleted Kind = "meshconfig-deleted"

	// MeshConfigUpdated is the type of announcement emitted when we observe an update to a Kubernetes MeshConfig
	MeshConfigUpdated Kind = "meshconfig-updated"

	// MeshRootCertificateAdded is the type of announcement emitted when we observe an addition of a Kubernetes MeshRootCertificate
	MeshRootCertificateAdded Kind = "meshrootcertificate-added"

	// MeshRootCertificateDeleted is the type of announcement emitted when we observe the deletion of a Kubernetes MeshRootCertificate
	MeshRootCertificateDeleted Kind = "meshrootcertificate-deleted"

	// MeshRootCertificateUpdated is the type of announcement emitted when we observe an update to a Kubernetes MeshRootCertificate
	MeshRootCertificateUpdated Kind = "meshrootcertificate-updated"

	// --- policy.openservicemesh.io API events

	// EgressAdded is the type of announcement emitted when we observe an addition of egresses.policy.openservicemesh.io
	EgressAdded Kind = "egress-added"

	// EgressDeleted the type of announcement emitted when we observe a deletion of egresses.policy.openservicemesh.io
	EgressDeleted Kind = "egress-deleted"

	// EgressUpdated is the type of announcement emitted when we observe an update to egresses.policy.openservicemesh.io
	EgressUpdated Kind = "egress-updated"

	// IngressBackendAdded is the type of announcement emitted when we observe an addition of ingressbackends.policy.openservicemesh.io
	IngressBackendAdded Kind = "ingressbackend-added"

	// IngressBackendDeleted the type of announcement emitted when we observe a deletion of ingressbackends.policy.openservicemesh.io
	IngressBackendDeleted Kind = "ingressbackend-deleted"

	// IngressBackendUpdated is the type of announcement emitted when we observe an update to ingressbackends.policy.openservicemesh.io
	IngressBackendUpdated Kind = "ingressbackend-updated"

	// RetryPolicyAdded is the type of announcement emitted when we observe an addition of retries.policy.openservicemesh.io
	RetryPolicyAdded Kind = "retry-added"

	// RetryPolicyDeleted the type of announcement emitted when we observe a deletion of retries.policy.openservicemesh.io
	RetryPolicyDeleted Kind = "retry-deleted"

	// RetryPolicyUpdated is the type of announcement emitted when we observe an update to retries.policy.openservicemesh.io
	RetryPolicyUpdated Kind = "retry-updated"

	// UpstreamTrafficSettingAdded is the type of announcement emitted when we observe an addition of upstreamtrafficsettings.policy.openservicemesh.io
	UpstreamTrafficSettingAdded Kind = "upstreamtrafficsetting-added"

	// UpstreamTrafficSettingDeleted is the type of announcement emitted when we observe a deletion of upstreamtrafficsettings.policy.openservicemesh.io
	UpstreamTrafficSettingDeleted Kind = "upstreamtrafficsetting-deleted"

	// UpstreamTrafficSettingUpdated is the type of announcement emitted when we observe an update of upstreamtrafficsettings.policy.openservicemesh.io
	UpstreamTrafficSettingUpdated Kind = "upstreamtrafficsetting-updated"
)

// Announcement is a struct for messages between various components of OSM signaling a need for a change in Envoy proxy configuration
type Announcement struct {
	Type               Kind
	ReferencedObjectID interface{}
}
