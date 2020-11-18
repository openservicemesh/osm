package announcements

// AnnouncementType is used to record the type of announcement
type AnnouncementType string

func (at AnnouncementType) String() string {
	return string(at)
}

const (
	// ScheduleProxyBroadcast is used by other modules to request the dispatcher to schedule a global proxy broadcast
	ScheduleProxyBroadcast AnnouncementType = "schedule-proxy-broadcast"

	// ProxyBroadcast is used to notify all Proxy streams that they need to trigger an update
	ProxyBroadcast AnnouncementType = "proxy-broadcast"

	// PodAdded is the type of announcement emitted when we observe an addition of a Kubernetes Pod
	PodAdded AnnouncementType = "pod-added"

	// PodDeleted the type of announcement emitted when we observe the deletion of a Kubernetes Pod
	PodDeleted AnnouncementType = "pod-deleted"

	// PodUpdated is the type of announcement emitted when we observe an update to a Kubernetes Pod
	PodUpdated AnnouncementType = "pod-updated"

	// ---

	// EndpointAdded is the type of announcement emitted when we observe an addition of a Kubernetes Endpoint
	EndpointAdded AnnouncementType = "endpoint-added"

	// EndpointDeleted the type of announcement emitted when we observe the deletion of a Kubernetes Endpoint
	EndpointDeleted AnnouncementType = "endpoint-deleted"

	// EndpointUpdated is the type of announcement emitted when we observe an update to a Kubernetes Endpoint
	EndpointUpdated AnnouncementType = "endpoint-updated"

	// ---

	// NamespaceAdded is the type of announcement emitted when we observe an addition of a Kubernetes Namespace
	NamespaceAdded AnnouncementType = "namespace-added"

	// NamespaceDeleted the type of announcement emitted when we observe the deletion of a Kubernetes Namespace
	NamespaceDeleted AnnouncementType = "namespace-deleted"

	// NamespaceUpdated is the type of announcement emitted when we observe an update to a Kubernetes Namespace
	NamespaceUpdated AnnouncementType = "namespace-updated"

	// ---

	// ServiceAdded is the type of announcement emitted when we observe an addition of a Kubernetes Service
	ServiceAdded AnnouncementType = "service-added"

	// ServiceDeleted the type of announcement emitted when we observe the deletion of a Kubernetes Service
	ServiceDeleted AnnouncementType = "service-deleted"

	// ServiceUpdated is the type of announcement emitted when we observe an update to a Kubernetes Service
	ServiceUpdated AnnouncementType = "service-updated"

	// ---

	// TrafficSplitAdded is the type of announcement emitted when we observe an addition of a Kubernetes TrafficSplit
	TrafficSplitAdded AnnouncementType = "trafficsplit-added"

	// TrafficSplitDeleted the type of announcement emitted when we observe the deletion of a Kubernetes TrafficSplit
	TrafficSplitDeleted AnnouncementType = "trafficsplit-deleted"

	// TrafficSplitUpdated is the type of announcement emitted when we observe an update to a Kubernetes TrafficSplit
	TrafficSplitUpdated AnnouncementType = "trafficsplit-updated"

	// ---

	// RouteGroupAdded is the type of announcement emitted when we observe an addition of a Kubernetes RouteGroup
	RouteGroupAdded AnnouncementType = "routegroup-added"

	// RouteGroupDeleted the type of announcement emitted when we observe the deletion of a Kubernetes RouteGroup
	RouteGroupDeleted AnnouncementType = "routegroup-deleted"

	// RouteGroupUpdated is the type of announcement emitted when we observe an update to a Kubernetes RouteGroup
	RouteGroupUpdated AnnouncementType = "routegroup-updated"

	// ---

	// TCPRouteAdded is the type of announcement emitted when we observe an addition of a Kubernetes TCPRoute
	TCPRouteAdded AnnouncementType = "tcproute-added"

	// TCPRouteDeleted the type of announcement emitted when we observe the deletion of a Kubernetes TCPRoute
	TCPRouteDeleted AnnouncementType = "tcproute-deleted"

	// TCPRouteUpdated is the type of announcement emitted when we observe an update to a Kubernetes TCPRoute
	TCPRouteUpdated AnnouncementType = "tcproute-updated"

	// ---

	// TrafficTargetAdded is the type of announcement emitted when we observe an addition of a Kubernetes TrafficTarget
	TrafficTargetAdded AnnouncementType = "traffictarget-added"

	// TrafficTargetDeleted the type of announcement emitted when we observe the deletion of a Kubernetes TrafficTarget
	TrafficTargetDeleted AnnouncementType = "traffictarget-deleted"

	// TrafficTargetUpdated is the type of announcement emitted when we observe an update to a Kubernetes TrafficTarget
	TrafficTargetUpdated AnnouncementType = "traffictarget-updated"

	// ---

	// BackpressureAdded is the type of announcement emitted when we observe an addition of a Kubernetes Backpressure
	BackpressureAdded AnnouncementType = "backpressure-added"

	// BackpressureDeleted the type of announcement emitted when we observe the deletion of a Kubernetes Backpressure
	BackpressureDeleted AnnouncementType = "backpressure-deleted"

	// BackpressureUpdated is the type of announcement emitted when we observe an update to a Kubernetes Backpressure
	BackpressureUpdated AnnouncementType = "backpressure-updated"

	// ---

	// ConfigMapAdded is the type of announcement emitted when we observe an addition of a Kubernetes ConfigMap
	ConfigMapAdded AnnouncementType = "configmap-added"

	// ConfigMapDeleted the type of announcement emitted when we observe the deletion of a Kubernetes ConfigMap
	ConfigMapDeleted AnnouncementType = "configmap-deleted"

	// ConfigMapUpdated is the type of announcement emitted when we observe an update to a Kubernetes ConfigMap
	ConfigMapUpdated AnnouncementType = "configmap-updated"

	// ---

	// IngressAdded is the type of announcement emitted when we observe an addition of a Kubernetes Ingress
	IngressAdded AnnouncementType = "ingress-added"

	// IngressDeleted the type of announcement emitted when we observe the deletion of a Kubernetes Ingress
	IngressDeleted AnnouncementType = "ingress-deleted"

	// IngressUpdated is the type of announcement emitted when we observe an update to a Kubernetes Ingress
	IngressUpdated AnnouncementType = "ingress-updated"
)

// Announcement is a struct for messages between various components of OSM signaling a need for a change in Envoy proxy configuration
type Announcement struct {
	Type               AnnouncementType
	ReferencedObjectID interface{}
}
