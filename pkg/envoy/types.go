// Package envoy implements utility routines related to Envoy proxy, and models an instance of a proxy
// to be able to generate XDS configurations for it.
package envoy

import (
	"github.com/openservicemesh/osm/pkg/logger"
	"github.com/openservicemesh/osm/pkg/utils"
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

// Short returns an abbreviated version of the TypeURI, which is easier to spot in logs and metrics.
func (t TypeURI) Short() string {
	return utils.GetLastChunkOfSlashed(t.String())
}

// IsWildcard returns true when the TypeURI is the one requesting ALL discovery types.
func (t TypeURI) IsWildcard() bool {
	return t == TypeWildcard
}

// ValidURI defines valid URIs
var ValidURI = map[string]TypeURI{
	string(TypeWildcard):           TypeWildcard,
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
	TypeSDS: "SDS",
	TypeCDS: "CDS",
	TypeLDS: "LDS",
	TypeRDS: "RDS",
	TypeEDS: "EDS",
}

// Envoy TypeURIs
const (
	// TypeWildcard is the Wildcard type URI.
	TypeWildcard TypeURI = ""

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

const (
	accessLogPath = "/dev/stdout"

	// localClusterSuffix is the tag to append to the local cluster name corresponding to a service cluster.
	// The local cluster refers to the cluster corresponding to the service the proxy is fronting, accessible over localhost by the proxy.
	localClusterSuffix = "-local"
)
