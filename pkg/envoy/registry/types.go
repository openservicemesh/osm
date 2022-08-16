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
	connectedProxies map[int64]*envoy.Proxy
}
