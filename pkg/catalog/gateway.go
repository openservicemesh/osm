package catalog

import (
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/identity"
)

// IsMultiClusterGateway checks if the ServiceIdentity belongs to the MultiClusterGateway.
// Only used if MultiClusterMode is enabled.
func (mc *MeshCatalog) IsMultiClusterGateway(svcID identity.ServiceIdentity) bool {
	sa := svcID.ToK8sServiceAccount()
	return mc.configurator.GetFeatureFlags().EnableMulticlusterMode &&
		envoy.ProxyKind(sa.Name) == envoy.KindGateway && sa.Namespace == mc.configurator.GetOSMNamespace()
}
