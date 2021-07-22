package multicluster

import (
	"github.com/google/uuid"
	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/envoy"
)

func GetMulticlusterGatewaySubjectCommonName(serviceAccount, namespace string) certificate.CommonName {
	gatewayUID := uuid.New()
	envoyType := envoy.KindMulticlusterGateway
	return envoy.NewXDSCertCommonName(gatewayUID, envoyType, serviceAccount, namespace)
}
