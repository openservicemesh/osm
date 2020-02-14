package envoy

type TypeURI string

const (
	TypeSDS TypeURI = "type.googleapis.com/envoy.api.v2.auth.Secret"
	TypeCDS TypeURI = "type.googleapis.com/envoy.api.v2.Cluster"
	TypeLDS TypeURI = "type.googleapis.com/envoy.api.v2.Listener"
	TypeRDS TypeURI = "type.googleapis.com/envoy.api.v2.RouteConfiguration"
	TypeEDS TypeURI = "type.googleapis.com/envoy.api.v2.ClusterLoadAssignment"
)
