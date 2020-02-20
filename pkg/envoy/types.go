package envoy

type TypeURI string

const (
	TypeSDS TypeURI = "type.googleapis.com/envoy.api.v2.auth.Secret"
	TypeCDS TypeURI = "type.googleapis.com/envoy.api.v2.Cluster"
	TypeLDS TypeURI = "type.googleapis.com/envoy.api.v2.Listener"
	TypeRDS TypeURI = "type.googleapis.com/envoy.api.v2.RouteConfiguration"
	TypeEDS TypeURI = "type.googleapis.com/envoy.api.v2.ClusterLoadAssignment"

	TransportSocketTLS = "envoy.transport_sockets.tls"
	CertificateName    = "server_cert"

	accessLogPath = "/dev/stdout"

	// cipher suites
	aes    = "ECDHE-ECDSA-AES128-GCM-SHA256"
	chacha = "ECDHE-ECDSA-CHACHA20-POLY1305"
)
