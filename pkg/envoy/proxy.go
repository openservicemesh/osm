package envoy

import (
	"net"
	"strings"

	"github.com/deislabs/smc/pkg/certificate"
	"github.com/deislabs/smc/pkg/endpoint"
	"github.com/deislabs/smc/pkg/utils"
)

const (
	dot = "."
)

// EnvoyProxy is a representation of an Envoy proxy connected to the xDS server.
// This should at some point have a 1:1 match to an Endpoint (which is a member of a meshed service).
type Proxy struct {
	certificate.CommonName
	net.IP
	ID string
}

// GetService implements Proxyer and determines the meshed service this endpoint should support based on the mTLS certificate.
// From "a.b.c" returns "b.c". By convention "a" is the ID of the proxy. Remaining "b.c" is the name of the service.
func (p Proxy) GetService() endpoint.ServiceName {
	cn := string(p.CommonName)
	dotCount := strings.Count(cn, dot)
	return endpoint.ServiceName(utils.GetLastNOfDotted(cn, dotCount))
}

// GetCommonName implements Proxyer and returns the Subject Common Name from the mTLS certificate of the Envoy proxy connected to xDS.
func (p Proxy) GetCommonName() certificate.CommonName {
	return p.CommonName
}

// GetIP implements Proxyer and returns the IP address of the Envoy proxy connected to xDS.
func (p Proxy) GetIP() net.IP {
	return p.IP
}

func (p Proxy) GetID() string {
	return p.ID
}

// NewProxy creates a new instance of an Envoy proxy connected to the xDS servers.
func NewProxy(cn certificate.CommonName, ip net.IP) Proxyer {
	return Proxy{
		CommonName: cn,
		IP:         ip,
		ID:         utils.NewUuidStr(),
	}
}
