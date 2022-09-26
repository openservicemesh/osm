package models

import (
	"fmt"
	"net"
	"time"

	"github.com/google/uuid"

	"github.com/openservicemesh/osm/pkg/identity"
)

// Proxy is a representation of an Envoy proxy connected to the xDS server.
// This should at some point have a 1:1 match to an Endpoint (which is a member of a meshed service).
type Proxy struct {
	// UUID of the proxy
	uuid.UUID

	Identity identity.ServiceIdentity

	net.Addr

	// The time this Proxy connected to the OSM control plane
	connectedAt time.Time
	// Connection ID is used to distinguish a single proxy that reconnects from the old proxy.
	// The one with the larger ID is the newer proxy.
	connectionID int64

	// kind is the proxy's kind (ex. sidecar, gateway)
	kind ProxyKind
}

func (p *Proxy) String() string {
	return fmt.Sprintf("[ProxyIdentity=%s], [ProxyUUID=%s]", p.Identity, p.UUID)
}

// GetConnectedAt returns the timestamp of when the given proxy connected to the control plane.
func (p *Proxy) GetConnectedAt() time.Time {
	return p.connectedAt
}

// GetConnectionID returns the connection ID of the proxy.
// Connection ID is used to distinguish a single proxy that reconnects from the old proxy.
// The one with the larger ID is the newer proxy.
// NOTE: it is not used properly in the old, StreamAggregatedResources, and only works properly for the SnapshotCache.
func (p *Proxy) GetConnectionID() int64 {
	return p.connectionID
}

// GetIP returns the IP address of the Envoy proxy connected to xDS.
func (p *Proxy) GetIP() net.Addr {
	return p.Addr
}

// Kind return the proxy's kind
func (p *Proxy) Kind() ProxyKind {
	return p.kind
}

// NewProxy creates a new instance of an Envoy proxy connected to the xDS servers.
func NewProxy(kind ProxyKind, uuid uuid.UUID, svcIdentity identity.ServiceIdentity, ip net.Addr, connectionID int64) *Proxy {
	return &Proxy{
		// Identity is of the form <name>.<namespace>.cluster.local
		Identity: svcIdentity,
		UUID:     uuid,

		Addr: ip,

		connectedAt:  time.Now(),
		connectionID: connectionID,

		kind: kind,
	}
}

// NewXDSCertCNPrefix returns a newly generated CommonName for a certificate of the form: <ProxyUUID>.<kind>.<identity>
// where identity itself is of the form <name>.<namespace>
func NewXDSCertCNPrefix(proxyUUID uuid.UUID, kind ProxyKind, si identity.ServiceIdentity) string {
	return fmt.Sprintf("%s.%s.%s", proxyUUID.String(), kind, si)
}
