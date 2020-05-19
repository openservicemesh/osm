package envoy

import (
	"github.com/open-service-mesh/osm/pkg/logger"
)

var (
	// XDSResponseOrder is the order in which we send xDS responses: CDS, EDS, LDS, RDS
	// See: https://github.com/envoyproxy/go-control-plane/issues/59
	XDSResponseOrder = []TypeURI{TypeCDS, TypeEDS, TypeLDS, TypeRDS, TypeSDS}

	log = logger.New("envoy")
)

// TypeURI is a string describing the Envoy xDS payload.
type TypeURI string

const (
	// TypeSDS is the SDS type URI.
	TypeSDS TypeURI = "type.googleapis.com/envoy.api.v2.auth.Secret"

	// TypeCDS is the CDS type URI.
	TypeCDS TypeURI = "type.googleapis.com/envoy.api.v2.Cluster"

	// TypeLDS is the LDS type URI.
	TypeLDS TypeURI = "type.googleapis.com/envoy.api.v2.Listener"

	// TypeRDS is the RDS type URI.
	TypeRDS TypeURI = "type.googleapis.com/envoy.api.v2.RouteConfiguration"

	// TypeEDS is the EDS type URI.
	TypeEDS TypeURI = "type.googleapis.com/envoy.api.v2.ClusterLoadAssignment"

	// TypeUpstreamTLSContext is an Envoy type URI.
	TypeUpstreamTLSContext TypeURI = "type.googleapis.com/envoy.api.v2.auth.UpstreamTlsContext"

	// TransportSocketTLS is an Envoy string constant.
	TransportSocketTLS = "envoy.transport_sockets.tls"

	accessLogPath = "/dev/stdout"

	//LocalClusterSuffix is the tag to append to local clusters
	LocalClusterSuffix = "-local"
)
