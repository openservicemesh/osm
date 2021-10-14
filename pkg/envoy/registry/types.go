package registry

import (
	"sync"
	"time"

	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/logger"
	"github.com/openservicemesh/osm/pkg/messaging"
)

var log = logger.New("proxy-registry")

// ProxyRegistry keeps track of Envoy proxies as they connect and disconnect
// from the control plane.
type ProxyRegistry struct {
	ProxyServiceMapper

	connectedProxies sync.Map

	// Maintain a mapping of pod UID to CN of the Envoy on the given pod
	podUIDToCN sync.Map

	// Maintain a mapping of pod UID to certificate SerialNumber of the Envoy on the given pod
	podUIDToCertificateSerialNumber sync.Map

	msgBroker *messaging.Broker
}

type connectedProxy struct {
	// Proxy which connected to the XDS control plane
	proxy *envoy.Proxy

	// When the proxy connected to the XDS control plane
	connectedAt time.Time
}
