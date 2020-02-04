package envoy

import (
	"net"

	"github.com/deislabs/smc/pkg/certificate"
	"github.com/deislabs/smc/pkg/endpoint"
)

// Proxyer is interface for a proxy or side-car connected to the service mesh control plane.
// This is strictly dealing with the control plane idea of "proxy". Not the data plane "endpoint".
type Proxyer interface {
	// GetService returns the service, which the process fronted by this proxy is a member of.
	GetService() endpoint.ServiceName

	// GetCommonName returns the Subject Common Name of the certificate assigned to this proxy.
	// This is a unique identifier for the proxy. Format is "<proxy-UUID>.<service-FQDN>"
	GetCommonName() certificate.CommonName

	// GetIP returns the IP address of the proxy.
	GetIP() net.IP

	GetID() string
}
