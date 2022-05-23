package secrets

import (
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/service"
)

const (
	// Separator is the separator between the prefix and the name of the certificate.
	Separator = ":"
)

// SDSCert is only used to interface the naming and related functions to Marshal/Unmarshal a resource name,
// this avoids having sprintf/parsing logic all over the place
type SDSCert interface {
	String() string
	validate() error
}

// SDSServiceCert represents an SDS Secret for mTLS, based on the local identity.
type SDSServiceCert struct {
	identity identity.ServiceIdentity
}

// SDSOutboundRootCert represents an SDS Secret for outbound mtls validation, based on service.
type SDSOutboundRootCert struct {
	service service.MeshService
}

// SDSInboundRootCert represents an SDS Secret for inbound mtls validation, based on identity.
type SDSInboundRootCert struct {
	identity identity.ServiceIdentity
}

// sdsCertType enums
const (
	// servicePrefix is the prefix for the service certificate resource name. Example: "service-cert:ns/name"
	servicePrefix = "service-cert"

	// outboundRootPrefix is the prefix for the mTLS root certificate resource name for upstream connectivity. Example: "root-cert-for-mtls-outbound:ns/name"
	outboundRootPrefix = "root-cert-for-mtls-outbound"

	// inboundRootPrefix is the prefix for the mTLS root certificate resource name for downstream connectivity. Example: "root-cert-for-mtls-inbound:ns/name"
	inboundRootPrefix = "root-cert-for-mtls-inbound"
)
