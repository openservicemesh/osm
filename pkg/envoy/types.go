package envoy

// TypeURI is a string describing the Envoy xDS payload.
type TypeURI string

// XDSResponseOrder is the order in which we send xDS responses: CDS, EDS, LDS, RDS
// See: https://github.com/envoyproxy/go-control-plane/issues/59
var XDSResponseOrder = []TypeURI{TypeCDS, TypeEDS, TypeLDS, TypeRDS, TypeSDS}

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

	// TransportSocketTLS is an Envoy string constant.
	TransportSocketTLS = "envoy.transport_sockets.tls"

	// CertificateName is a string constant for a certificate name.
	CertificateName = "server_cert"

	accessLogPath = "/dev/stdout"

	// cipher suites
	aes    = "ECDHE-ECDSA-AES128-GCM-SHA256"
	chacha = "ECDHE-ECDSA-CHACHA20-POLY1305"
)
