package multicluster

import (
	"github.com/google/uuid"

	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/identity"
)

// GetMulticlusterGatewaySubjectCommonName creates a unique certificate.CommonName
// specifically for a Multicluster Gateway. Each gateway will have its own unique
// cert. The kind of Envoy (gateway) is encoded in the cert CN by convention.
func GetMulticlusterGatewaySubjectCommonName(serviceAccount, namespace string) string {
	gatewayUID := uuid.New()
	envoyType := envoy.KindGateway
	return envoy.NewXDSCertCNPrefix(gatewayUID, envoyType, identity.New(serviceAccount, namespace))
}
