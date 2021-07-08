package catalog

import (
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/identity"
)

// isOSMGateway checks if the ServiceIdentity belongs to the MultiClusterGateway.
// Only used if MultiClusterMode is enabled.
// TODO(steeling): This doesn't work, since the identity is not set to kind-gateway.
func (mc *MeshCatalog) isOSMGateway(svcIdentity identity.ServiceIdentity) bool {
	sa := svcIdentity.ToK8sServiceAccount()
	return mc.configurator.GetFeatureFlags().EnableMulticlusterMode &&
		envoy.ProxyKind(sa.Name) == envoy.KindGateway && sa.Namespace == mc.configurator.GetOSMNamespace()
}
