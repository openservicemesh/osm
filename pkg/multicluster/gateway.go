package multicluster

import (
	"github.com/google/uuid"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/envoy"
)

// GetMulticlusterGatewaySubjectCommonName creates a unique certificate.CommonName
// specifically for a Multicluster Gateway. Each gateway will have its own unique
// cert. The kind of Envoy (gateway) is encoded in the cert CN by convention.
func GetMulticlusterGatewaySubjectCommonName(serviceAccount, namespace string) certificate.CommonName {
	gatewayUID := uuid.New()
	envoyType := envoy.KindGateway
	return envoy.NewXDSCertCommonName(gatewayUID, envoyType, serviceAccount, namespace)
}
