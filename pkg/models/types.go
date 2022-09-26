package models

// ProxyKind is the type used to define the proxy's kind
type ProxyKind string

const (
	// KindSidecar implies the proxy is a sidecar
	KindSidecar ProxyKind = "sidecar"
)
