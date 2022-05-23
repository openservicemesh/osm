package secrets

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/types"

	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/service"
)

const (
	// namespaceNameSeparator used upon marshalling/unmarshalling MeshService to a string
	// or viceversa
	namespaceNameSeparator = "/"
	identitySeparator      = "."
)

// UnmarshalSDSCert parses the SDS resource name and returns an SDSCert object and an error if any
// Examples:
// 1. Unmarshalling 'service-cert:foo/bar' returns SDSCert{CertType: service-cert, Name: foo/bar}, nil
// 2. Unmarshalling 'root-cert-for-mtls-inbound:foo/bar' returns SDSCert{CertType: root-cert-for-mtls-inbound, Name: foo/bar}, nil
// 3. Unmarshalling 'invalid-cert' returns nil, error
func UnmarshalSDSCert(str string) (SDSCert, error) {
	var ret SDSCert
	// Check separators, ignore empty string fields
	slices := strings.Split(str, Separator)
	if len(slices) != 2 {
		return nil, errInvalidCertFormat
	}

	// Make sure the slices are not empty. Split might actually leave empty slices.

	if slices[0] == "" || slices[1] == "" {
		return nil, errInvalidCertFormat
	}

	switch slices[0] {
	case servicePrefix:
		identity := identity.ServiceIdentity(slices[1])
		ret = &SDSServiceCert{identity: identity}
	case inboundRootPrefix:
		identity := identity.ServiceIdentity(slices[1])
		ret = &SDSInboundRootCert{identity: identity}
	case outboundRootPrefix:
		slices = strings.Split(slices[1], namespaceNameSeparator)
		if len(slices) != 2 {
			return nil, fmt.Errorf("bad cert format for for: %s", str)
		}
		ret = &SDSOutboundRootCert{service: service.MeshService{
			Namespace: slices[0],
			Name:      slices[1],
		}}
	default:
		return nil, fmt.Errorf("unknown cert type for: %s", str)
	}
	return ret, ret.validate()
}

func (sdsc *SDSServiceCert) String() string {
	return fmt.Sprintf("%s%s%s", servicePrefix, Separator, sdsc.identity.String())
}

func (sdsc *SDSInboundRootCert) String() string {
	return fmt.Sprintf("%s%s%s", inboundRootPrefix, Separator, sdsc.identity.String())
}

func (sdsc *SDSOutboundRootCert) String() string {
	return fmt.Sprintf("%s%s%s", outboundRootPrefix, Separator, sdsc.service.String())
}

func (sdsc *SDSServiceCert) validate() error {
	return validateIdentity(sdsc.identity)
}

func (sdsc *SDSInboundRootCert) validate() error {
	return validateIdentity(sdsc.identity)
}

func (sdsc *SDSOutboundRootCert) validate() error {
	if sdsc.service.Name == "" || sdsc.service.Namespace == "" {
		return fmt.Errorf("invalid mesh service: %s", sdsc.service.String())
	}
	return nil
}

func validateIdentity(si identity.ServiceIdentity) error {
	name, remaining, _ := strings.Cut(si.String(), identitySeparator)
	namespace, trustDomain, _ := strings.Cut(remaining, identitySeparator)
	if name == "" || namespace == "" || trustDomain == "" {
		return fmt.Errorf("invalid identity: %s", si.String())
	}
	return nil
}

// GetMeshService unmarshals a NamespaceService type from a SDSCert name
func (sdsc *SDSOutboundRootCert) GetMeshService() service.MeshService {
	return sdsc.service
}

// GetServiceIdentity unmarshals a K8sServiceAccount type from a SDSCert name
func (sdsc *SDSInboundRootCert) GetServiceIdentity() identity.ServiceIdentity {
	return sdsc.identity
}

// GetServiceIdentity unmarshals a K8sServiceAccount type from a SDSCert name
func (sdsc *SDSServiceCert) GetServiceIdentity() identity.ServiceIdentity {
	return sdsc.identity
}

// GetSDSServiceCertForIdentity returns an SDSServiceCert object for a given ServiceIdentity
func GetSDSServiceCertForIdentity(si identity.ServiceIdentity) *SDSServiceCert {
	return &SDSServiceCert{identity: si}
}

// GetSDSInboundRootCertForService returns an SDSInboundRootCert object for a given ServiceIdentity
func GetSDSInboundRootCertForIdentity(si identity.ServiceIdentity) *SDSInboundRootCert {
	return &SDSInboundRootCert{identity: si}
}

// GetSDSOutboundRootCertForMeshService returns an SDSOutboundRootCert object for a given MeshService
func GetSDSOutboundRootCertForService(ms service.MeshService) *SDSOutboundRootCert {
	return &SDSOutboundRootCert{service: ms}
}

// GetSDSOutboundRootCertForNamespacedName returns an SDSOutboundRootCert object for a given NamespacedName
func GetSDSOutboundRootCertForNamespacedName(name *types.NamespacedName) *SDSOutboundRootCert {
	return &SDSOutboundRootCert{
		service: service.MeshService{
			Name:      name.Name,
			Namespace: name.Namespace,
		},
	}
}
