package service

import (
	"fmt"
	"strings"

	"github.com/jinzhu/copier"
)

// CertType is a type of a certificate requested by an Envoy proxy via SDS.
type CertType string

// Direction is a type to identify TLS certificate connectivity direction.
type Direction string

// CertResource is only used to interface the naming and related functions to Marshal/Unmarshal a resource name,
// this avoids having sprintf/parsing logic all over the place
type CertResource struct {
	// Service is a namespaced service struct
	Service NamespacedService
	// CertType is the certificate type
	CertType CertType
}

func (ct CertType) String() string {
	return string(ct)
}

const (
	// ServiceCertType is the prefix for the service certificate resource name. Example: "service-cert:webservice"
	ServiceCertType CertType = "service-cert"

	// RootCertTypeForMTLSOutbound is the prefix for the mTLS root certificate resource name for upstream connectivity. Example: "root-cert-for-mtls-outbound:webservice"
	RootCertTypeForMTLSOutbound CertType = "root-cert-for-mtls-outbound"

	// RootCertTypeForMTLSInbound is the prefix for the mTLS root certificate resource name for downstream connectivity. Example: "root-cert-for-mtls-inbound:webservice"
	RootCertTypeForMTLSInbound CertType = "root-cert-for-mtls-inbound"

	// RootCertTypeForHTTPS is the prefix for the HTTPS root certificate resource name. Example: "root-cert-https:webservice"
	RootCertTypeForHTTPS CertType = "root-cert-https"

	// Outbound refers to Envoy upstream connectivity direction for TLS certs
	Outbound Direction = "outbound"

	// Inbound refers to Envoy downstream connectivity direction for TLS certs
	Inbound Direction = "inbound"

	// NoDirection for resources that do not specify a direction implied with the cert resource type
	NoDirection Direction = "no-direction"

	// Separator is the separator between the prefix and the name of the certificate.
	certResSeparator = ":"
)

// Defines valid cert types
var validCertTypes = map[CertType]interface{}{
	ServiceCertType:             nil,
	RootCertTypeForMTLSOutbound: nil,
	RootCertTypeForMTLSInbound:  nil,
	RootCertTypeForHTTPS:        nil,
}

// UnmarshalCertResource parses and returns Certificate type and Namespaced Service name given a
// correctly formatted string, otherwise returns error
func UnmarshalCertResource(str string) (*CertResource, error) {
	var svc *NamespacedService
	var ret CertResource

	// Check separators, ignore empty string fields
	slices := strings.Split(str, certResSeparator)
	if len(slices) != 2 {
		return nil, errInvalidCertFormat
	}

	// Make sure the slices are not empty. Split might actually leave empty slices.
	for _, sep := range slices {
		if len(sep) == 0 {
			return nil, errInvalidCertFormat
		}
	}

	// Check valid certType
	ret.CertType = CertType(slices[0])
	if _, ok := validCertTypes[ret.CertType]; !ok {
		return nil, errInvalidCertFormat
	}

	// Check valid namespace'd service name
	svc, err := UnmarshalNamespacedService(slices[1])
	if err != nil {
		return nil, err
	}
	err = copier.Copy(&ret.Service, &svc)
	if err != nil {
		return nil, err
	}

	return &ret, nil

}

// String is a common facility/interface to generate a string resource name out of a SDSCert
// This is to keep the sprintf logic and/or separators used agnostic to other modules
func (sdsc CertResource) String() string {
	return fmt.Sprintf("%s%s%s",
		sdsc.CertType.String(),
		certResSeparator,
		sdsc.Service.String())
}

// Direction returns direction (Direction type) of a certificate, or error
// if the specific type has not direction implied
func (sdsc CertResource) Direction() Direction {
	switch sdsc.CertType {
	case RootCertTypeForMTLSOutbound:
		return Outbound
	case RootCertTypeForMTLSInbound:
		return Inbound
	default:
		return NoDirection
	}
}

func (dir Direction) String() string {
	return string(dir)
}
