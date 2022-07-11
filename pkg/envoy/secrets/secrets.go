package secrets

import (
	"strings"

	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/logger"
	"github.com/openservicemesh/osm/pkg/service"
)

const (
	// namespaceNameSeparator used upon marshalling/unmarshalling MeshService to a string
	// or viceversa
	namespaceNameSeparator = "/"
)

var log = logger.New("secrets")

// UnmarshalSDSCert parses the SDS resource name and returns an SDSCert object and an error if any
// Examples:
// 1. Unmarshalling 'service-cert:foo/bar' returns SDSCert{CertType: service-cert, Name: foo/bar}, nil
// 2. Unmarshalling 'root-cert-for-mtls-inbound:foo/bar' returns SDSCert{CertType: root-cert-for-mtls-inbound, Name: foo/bar}, nil
// 3. Unmarshalling 'invalid-cert' returns nil, error
func UnmarshalSDSCert(str string) (*SDSCert, error) {
	var ret SDSCert

	// Check separators, ignore empty string fields
	slices := strings.Split(str, Separator)
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
	ret.CertType = SDSCertType(slices[0])
	if _, ok := validCertTypes[ret.CertType]; !ok {
		return nil, errInvalidCertFormat
	}

	ret.Name = slices[1]

	return &ret, nil
}

// GetMeshService unmarshals a NamespaceService type from a SDSCert name
func (sdsc *SDSCert) GetMeshService() (*service.MeshService, error) {
	slices := strings.Split(sdsc.Name, namespaceNameSeparator)
	if len(slices) != 2 {
		return nil, errInvalidMeshServiceFormat
	}

	// Make sure the slices are not empty. Split might actually leave empty slices.
	if slices[0] == "" || slices[1] == "" {
		return nil, errInvalidMeshServiceFormat
	}

	ms := service.MeshService{
		Namespace: slices[0],
		Name:      slices[1],
	}

	return &ms, nil
}

// GetK8sServiceAccount unmarshals a K8sServiceAccount type from a SDSCert name
func (sdsc *SDSCert) GetK8sServiceAccount() (*identity.K8sServiceAccount, error) {
	slices := strings.Split(sdsc.Name, namespaceNameSeparator)
	if len(slices) != 2 {
		log.Error().Msgf("Error converting Service Account %s from string to K8sServiceAccount", sdsc.Name)
		return nil, errInvalidNamespacedServiceStringFormat
	}

	// Make sure the slices are not empty. Split might actually leave empty slices.
	if slices[0] == "" || slices[1] == "" {
		log.Error().Msgf("Error converting Service Account %s from string to K8sServiceAccount", sdsc.Name)
		return nil, errInvalidNamespacedServiceStringFormat
	}

	return &identity.K8sServiceAccount{
		Namespace: slices[0],
		Name:      slices[1],
	}, nil
}

// GetSecretNameForIdentity returns the SDS secret name corresponding to the given ServiceIdentity
func GetSecretNameForIdentity(si identity.ServiceIdentity) string {
	// TODO(draychev): The cert names can be redone to move away from using "namespace/name" format [https://github.com/openservicemesh/osm/issues/2218]
	// Currently this will be: "service-cert:default/bookbuyer"
	return si.ToK8sServiceAccount().String()
}
