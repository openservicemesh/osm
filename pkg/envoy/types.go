package envoy

import (
	"net"

	"github.com/deislabs/smc/pkg/certificate"
	"github.com/deislabs/smc/pkg/endpoint"
)

const (
	baseURI = "type.googleapis.com/envoy.api.v2."
	TypeCDS = baseURI + "Cluster"
	TypeLDS = baseURI + "Listener"
	TypeRDS = baseURI + "RouteConfiguration"
	TypeSDS = baseURI + "auth.Secret"
	TypeEDS = baseURI + "ClusterLoadAssignment"
)

type ProxyID string

func (id ProxyID) String() string {
	return string(id)
}

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

	// GetID returns the UUID assigned to the proxy connected to the control plane
	GetID() ProxyID

	// GetAnnouncementsChannel returns the announcement channel the proxy is listening on
	GetAnnouncementsChannel() <-chan interface{}
}
