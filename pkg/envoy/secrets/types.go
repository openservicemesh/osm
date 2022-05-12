package secrets

import "fmt"

const (
	// Separator is the separator between the prefix and the name of the certificate.
	Separator = ":"
)

// SDSCertType is a type of a certificate requested by an Envoy proxy via SDS.
type SDSCertType string

// SDSCert is only used to interface the naming and related functions to Marshal/Unmarshal a resource name,
// this avoids having sprintf/parsing logic all over the place
type SDSCert struct {
	// Name is the name of the SDS secret for the certificate
	Name string

	// CertType is the certificate type
	CertType SDSCertType
}

// String is a common facility/interface to generate a string resource name out of a SDSCert
// This is to keep the sprintf logic and/or separators used agnostic to other modules
func (sdsc SDSCert) String() string {
	return fmt.Sprintf("%s%s%s",
		sdsc.CertType.String(),
		Separator,
		sdsc.Name)
}

func (ct SDSCertType) String() string {
	return string(ct)
}

// SDSCertType enums
const (
	// ServiceCertType is the prefix for the service certificate resource name. Example: "service-cert:ns/name"
	ServiceCertType SDSCertType = "service-cert"

	// RootCertTypeForMTLSOutbound is the prefix for the mTLS root certificate resource name for upstream connectivity. Example: "root-cert-for-mtls-outbound:ns/name"
	RootCertTypeForMTLSOutbound SDSCertType = "root-cert-for-mtls-outbound"

	// RootCertTypeForMTLSInbound is the prefix for the mTLS root certificate resource name for downstream connectivity. Example: "root-cert-for-mtls-inbound:ns/name"
	RootCertTypeForMTLSInbound SDSCertType = "root-cert-for-mtls-inbound"
)

// Defines valid cert types
var validCertTypes = map[SDSCertType]struct{}{
	ServiceCertType:             {},
	RootCertTypeForMTLSOutbound: {},
	RootCertTypeForMTLSInbound:  {},
}
