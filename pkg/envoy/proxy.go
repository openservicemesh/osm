package envoy

import (
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/deislabs/smc/pkg/certificate"
	"github.com/deislabs/smc/pkg/endpoint"
	"github.com/deislabs/smc/pkg/utils"
)

const (
	dot = "."
)

// Proxy is a representation of an Envoy proxy connected to the xDS server.
// This should at some point have a 1:1 match to an Endpoint (which is a member of a meshed service).
type Proxy struct {
	certificate.CommonName
	net.IP
	ServiceName   endpoint.NamespacedService
	announcements chan interface{}

	lastSentVersion    map[TypeURI]uint64
	lastAppliedVersion map[TypeURI]uint64
	lastNonce          map[TypeURI]string
}

// SetLastAppliedVersion records the version of the given Envoy proxy that was last acknowledged.
func (p *Proxy) SetLastAppliedVersion(typeURI TypeURI, version uint64) {
	p.lastAppliedVersion[typeURI] = version
}

// GetLastAppliedVersion returns the last version sucessfully applied to the given Envoy proxy.
func (p Proxy) GetLastAppliedVersion(typeURI TypeURI) uint64 {
	return p.lastAppliedVersion[typeURI]
}

// GetLastSentVersion returns the last sent version.
func (p Proxy) GetLastSentVersion(typeURI TypeURI) uint64 {
	return p.lastSentVersion[typeURI]
}

// IncrementLastSentVersion incremens last sent version.
func (p *Proxy) IncrementLastSentVersion(typeURI TypeURI) uint64 {
	p.lastSentVersion[typeURI]++
	return p.GetLastSentVersion(typeURI)
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

// String returns the CommonName of the proxy.
func (p Proxy) String() string {
	return string(p.GetCommonName())
}

// GetService determines the meshed service this endpoint should support based on the mTLS certificate.
// From "a.b.c" returns "b.c". By convention "a" is the ID of the proxy. Remaining "b.c" is the name of the service.
func (p Proxy) GetService() endpoint.NamespacedService {
	return p.ServiceName
}

// GetCommonName returns the Subject Common Name from the mTLS certificate of the Envoy proxy connected to xDS.
func (p Proxy) GetCommonName() certificate.CommonName {
	return p.CommonName
}

// GetIP returns the IP address of the Envoy proxy connected to xDS.
func (p Proxy) GetIP() net.IP {
	return p.IP
}

// GetAnnouncementsChannel returns the announcement channel for the given Envoy proxy.
func (p Proxy) GetAnnouncementsChannel() chan interface{} {
	return p.announcements
}

// NewProxy creates a new instance of an Envoy proxy connected to the xDS servers.
func NewProxy(cn certificate.CommonName, ip net.IP) *Proxy {
	dotCount := strings.Count(string(cn), dot)
	return &Proxy{
		CommonName:         cn,
		IP:                 ip,
		ServiceName:        endpoint.NamespacedService(getNamespacedService(utils.GetFirstNOfDotted(string(cn), dotCount))),
		announcements:      make(chan interface{}),
		lastNonce:          make(map[TypeURI]string),
		lastSentVersion:    make(map[TypeURI]uint64),
		lastAppliedVersion: make(map[TypeURI]uint64),
	}
}
