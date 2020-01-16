package envoy

import (
	"net"

	"github.com/deislabs/smc/pkg/certificate"
	"github.com/deislabs/smc/pkg/endpoint"
)

// Proxyer is interface for a proxy or side-car connected to the service mesh control plane.
// This is strictly dealing with the control plane idea of "proxy". Not the data plane "endpoint".
type Proxyer interface {
	GetService() endpoint.ServiceName
	GetCommonName() certificate.CommonName
	GetIP() net.IP
}
