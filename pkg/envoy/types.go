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

func (t TypeURI) String() string {
	return string(t)
}

// ValidURI defines valid URIs
var ValidURI = map[string]TypeURI{
	string(TypeSDS):                TypeSDS,
	string(TypeCDS):                TypeCDS,
	string(TypeLDS):                TypeLDS,
	string(TypeRDS):                TypeRDS,
	string(TypeEDS):                TypeEDS,
	string(TypeUpstreamTLSContext): TypeUpstreamTLSContext,
	string(TypeZipkinConfig):       TypeZipkinConfig,
}

const (
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

	accessLogPath = "/dev/stdout"

	//LocalClusterSuffix is the tag to append to local clusters
	LocalClusterSuffix = "-local"
)
