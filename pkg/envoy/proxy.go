package envoy

import (
	"fmt"
	"net"
	"time"

	mapset "github.com/deckarep/golang-set"
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
	// NOTE: it is not used properly in the old, StreamAggregatedResources, and only works properly for the SnapshotCache.
	connectionID int64

	lastSentVersion    map[TypeURI]uint64
	lastAppliedVersion map[TypeURI]uint64
	lastNonce          map[TypeURI]string

	// Contains the last resource names sent for a given proxy and TypeURL
	lastxDSResourcesSent map[TypeURI]mapset.Set

	// Contains the last requested resource names (and therefore, subscribed) for a given TypeURI
	subscribedResources map[TypeURI]mapset.Set

	// kind is the proxy's kind (ex. sidecar, gateway)
	kind ProxyKind
}

func (p *Proxy) String() string {
	return fmt.Sprintf("[ProxyIdentity=%s], [ProxyUUID=%s]", p.Identity, p.UUID)
}

// SetLastAppliedVersion records the version of the given Envoy proxy that was last acknowledged.
func (p *Proxy) SetLastAppliedVersion(typeURI TypeURI, version uint64) {
	p.lastAppliedVersion[typeURI] = version
}

// GetLastAppliedVersion returns the last version successfully applied to the given Envoy proxy.
func (p *Proxy) GetLastAppliedVersion(typeURI TypeURI) uint64 {
	return p.lastAppliedVersion[typeURI]
}

// GetLastSentVersion returns the last sent version.
func (p *Proxy) GetLastSentVersion(typeURI TypeURI) uint64 {
	return p.lastSentVersion[typeURI]
}

// IncrementLastSentVersion increments last sent version.
func (p *Proxy) IncrementLastSentVersion(typeURI TypeURI) uint64 {
	p.lastSentVersion[typeURI]++
	return p.GetLastSentVersion(typeURI)
}

// SetLastSentVersion records the version of the given config last sent to the proxy.
func (p *Proxy) SetLastSentVersion(typeURI TypeURI, ver uint64) {
	p.lastSentVersion[typeURI] = ver
}

// GetLastSentNonce returns last sent nonce.
func (p *Proxy) GetLastSentNonce(typeURI TypeURI) string {
	nonce, ok := p.lastNonce[typeURI]
	if !ok {
		p.lastNonce[typeURI] = ""
		return ""
	}
	return nonce
}

// SetNewNonce sets and returns a new nonce.
func (p *Proxy) SetNewNonce(typeURI TypeURI) string {
	p.lastNonce[typeURI] = fmt.Sprintf("%d", time.Now().UnixNano())
	return p.lastNonce[typeURI]
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

// GetLastResourcesSent returns a set of resources last sent for a proxy givne a TypeURL
// If none were sent, empty set is returned
func (p *Proxy) GetLastResourcesSent(typeURI TypeURI) mapset.Set {
	sentResources, ok := p.lastxDSResourcesSent[typeURI]
	if !ok {
		return mapset.NewSet()
	}
	return sentResources
}

// SetLastResourcesSent sets the last sent resources given a proxy for a TypeURL
func (p *Proxy) SetLastResourcesSent(typeURI TypeURI, resourcesSet mapset.Set) {
	p.lastxDSResourcesSent[typeURI] = resourcesSet
}

// GetSubscribedResources returns a set of resources subscribed for a proxy given a TypeURL
// If none were subscribed, empty set is returned
func (p *Proxy) GetSubscribedResources(typeURI TypeURI) mapset.Set {
	sentResources, ok := p.subscribedResources[typeURI]
	if !ok {
		return mapset.NewSet()
	}
	return sentResources
}

// SetSubscribedResources sets the input resources as subscribed resources given a proxy for a TypeURL
func (p *Proxy) SetSubscribedResources(typeURI TypeURI, resourcesSet mapset.Set) {
	p.subscribedResources[typeURI] = resourcesSet
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

		lastNonce:            make(map[TypeURI]string),
		lastSentVersion:      make(map[TypeURI]uint64),
		lastAppliedVersion:   make(map[TypeURI]uint64),
		lastxDSResourcesSent: make(map[TypeURI]mapset.Set),
		subscribedResources:  make(map[TypeURI]mapset.Set),

		kind: kind,
	}
}

// NewXDSCertCNPrefix returns a newly generated CommonName for a certificate of the form: <ProxyUUID>.<kind>.<identity>
// where identity itself is of the form <name>.<namespace>
func NewXDSCertCNPrefix(proxyUUID uuid.UUID, kind ProxyKind, si identity.ServiceIdentity) string {
	return fmt.Sprintf("%s.%s.%s", proxyUUID.String(), kind, si)
}
