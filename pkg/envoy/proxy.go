package envoy

import (
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
	endpoint.ServiceName
	announcements chan interface{}

	LastUpdated time.Time
	LastVersion uint64
	LastNonce   string
}

// GetService determines the meshed service this endpoint should support based on the mTLS certificate.
// From "a.b.c" returns "b.c". By convention "a" is the ID of the proxy. Remaining "b.c" is the name of the service.
func (p Proxy) GetService() endpoint.ServiceName {
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
		CommonName:    cn,
		IP:            ip,
		ServiceName:   endpoint.ServiceName(utils.GetLastNOfDotted(string(cn), dotCount)),
		announcements: make(chan interface{}),
	}
}
