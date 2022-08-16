package registry

import (
	"sync"

	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/logger"
	"github.com/openservicemesh/osm/pkg/messaging"
)

var log = logger.New("proxy-registry")

// ProxyRegistry keeps track of Envoy proxies as they connect and disconnect
// from the control plane.
type ProxyRegistry struct {
	ProxyServiceMapper

	msgBroker *messaging.Broker

	mu               sync.Mutex
	connectedProxies map[string]*envoy.Proxy
}

// A simple interface to release certificates. Created to abstract the certificate.Manager struct for testing purposes.
type certificateReleaser interface {
	ReleaseCertificate(key string)
}
