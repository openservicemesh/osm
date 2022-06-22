// Package envoy implements utility routines related to Envoy proxy, and models an instance of a proxy
// to be able to generate XDS configurations for it.
package envoy

import (
	"github.com/openservicemesh/osm/pkg/logger"
)

var (
	// XDSResponseOrder is the order in which we send xDS responses: CDS, EDS, LDS, RDS
	// See: https://github.com/envoyproxy/go-control-plane/issues/59
	XDSResponseOrder = []TypeURI{TypeCDS, TypeEDS, TypeLDS, TypeRDS, TypeSDS}

	log = logger.New("envoy")
)

// TypeURI is a string describing the Envoy xDS payload.
type TypeURI string

// IsWildcardTypeURI returns if a given TypeURI is an expected wildcard TypeURI or not.
// XDS proto defines general client behavior as:
// "Envoy will always use wildcard subscriptions for Listener and Cluster resources"
// https://www.envoyproxy.io/docs/envoy/latest/api-docs/xds_protocol#client-behavior
func IsWildcardTypeURI(t TypeURI) bool {
	return t == TypeCDS || t == TypeLDS
}

func (t TypeURI) String() string {
	return string(t)
}

// Short returns an abbreviated version of the TypeURI, which is easier to spot in logs and metrics.
func (t TypeURI) Short() string {
	return XDSShortURINames[t]
}

// ValidURI defines valid URIs
var ValidURI = map[string]TypeURI{
	string(TypeEmptyURI):           TypeEmptyURI,
	string(TypeSDS):                TypeSDS,
	string(TypeCDS):                TypeCDS,
	string(TypeLDS):                TypeLDS,
	string(TypeRDS):                TypeRDS,
	string(TypeEDS):                TypeEDS,
	string(TypeUpstreamTLSContext): TypeUpstreamTLSContext,
	string(TypeZipkinConfig):       TypeZipkinConfig,
}

// XDSShortURINames are shortened versions of the URI types
var XDSShortURINames = map[TypeURI]string{
	TypeEmptyURI: "EmptyURI",
	TypeSDS:      "SDS",
	TypeCDS:      "CDS",
	TypeLDS:      "LDS",
	TypeRDS:      "RDS",
	TypeEDS:      "EDS",
}

// Envoy TypeURIs
const (
	// TypeEmptyURI is an Empty URI type representation
	TypeEmptyURI TypeURI = ""

	// TypeSDS is the SDS type URI.
	TypeSDS TypeURI = "type.googleapis.com/envoy.extensions.transport_sockets.tls.v3.Secret"

	// TypeCDS is the CDS type URI.
	TypeCDS TypeURI = "type.googleapis.com/envoy.config.cluster.v3.Cluster"

	// TypeLDS is the LDS type URI.
	TypeLDS TypeURI = "type.googleapis.com/envoy.config.listener.v3.Listener"

	// TypeRDS is the RDS type URI.
	TypeRDS TypeURI = "type.googleapis.com/envoy.config.route.v3.RouteConfiguration"

	// TypeEDS is the EDS type URI.
	TypeEDS TypeURI = "type.googleapis.com/envoy.config.endpoint.v3.ClusterLoadAssignment"

	// TypeUpstreamTLSContext is an Envoy type URI.
	TypeUpstreamTLSContext TypeURI = "type.googleapis.com/envoy.extensions.transport_sockets.tls.v3.UpstreamTlsContext"

	// TypeZipkinConfig is an Envoy type URI.
	TypeZipkinConfig TypeURI = "type.googleapis.com/envoy.config.trace.v3.ZipkinConfig"

	// TypeADS is not actually used by Envoy - but useful within OSM for logging
	TypeADS TypeURI = "ADS"
)

// Filter names - can be any name (not used by Envoy to determine the filter to use)
// *Note: HTTP typed filters referenced in RDS require a wellknown name
const (
	// HTTP filters
	HTTPConnectionManagerFilterName = "http_connection_manager"
	HTTPRouterFilterName            = "http_router"
	HTTPLuaFilterName               = "http_lua"

	HTTPExtAuthzFilterName    = "http_external_authz"
	HTTPHealthCheckFilterName = "http_health_check"

	// The HTTP typed filters referenced in the RDS configuration still need to
	// use wellknown names. These filters are configured as a map where the key is
	// the filter name and value is the marshalled filter config.
	// See https://github.com/envoyproxy/envoy/issues/21759#issuecomment-1163570994
	HTTPRBACFilterName           = "envoy.filters.http.rbac"
	HTTPLocalRateLimitFilterName = "envoy.filters.http.local_ratelimit"

	// Network (L4) filters
	TCPProxyFilterName         = "tcp_proxy"
	L4LocalRateLimitFilterName = "l4_local_rate_limit"
	L4RBACFilterName           = "l4_rbac"

	// Listener filters
	OriginalDstFilterName   = "original_dst"
	TLSInspectorFilterName  = "tls_inspector"
	HTTPInspectorFilterName = "http_inspector"
)

// Filter TypeURLs - used by Envoy to determine the filter to use
const (
	HTTPRouterFilterTypeURL    = "type.googleapis.com/envoy.extensions.filters.http.router.v3.Router"
	HTTPRBACFilterTypeURL      = "type.googleapis.com/envoy.extensions.filters.http.rbac.v3.RBAC"
	OriginalDstFilterTypeURL   = "type.googleapis.com/envoy.extensions.filters.listener.original_dst.v3.OriginalDst"
	TLSInspectorFilterTypeURL  = "type.googleapis.com/envoy.extensions.filters.listener.tls_inspector.v3.TlsInspector"
	HTTPInspectorFilterTypeURL = "type.googleapis.com/envoy.extensions.filters.listener.http_inspector.v3.HttpInspector"
)

const (
	// EnvoyActiveHealthCheckPath is the HTTP endpoint to be used to receive
	// active health checks.
	EnvoyActiveHealthCheckPath = "/healthz/osm"

	// EnvoyActiveHealthCheckHeaderKey is the HTTP header key used to identify
	// active health check traffic.
	EnvoyActiveHealthCheckHeaderKey = "x-osm-envoy-healthcheck"
)

// ProxyKind is the type used to define the proxy's kind
type ProxyKind string

const (
	// KindSidecar implies the proxy is a sidecar
	KindSidecar ProxyKind = "sidecar"
)
