package secrets

import (
	"fmt"

	"github.com/openservicemesh/osm/pkg/identity"
)

const (
	// NameForMTLSInbound is the name of the sds secret for inbound mTLS validation.
	NameForMTLSInbound = "root-cert-for-mtls-inbound"
)

// NameForIdentity returns the SDS secret name corresponding to the given ServiceIdentity
func NameForIdentity(si identity.ServiceIdentity) string {
	// TODO(draychev): The cert names can be redone to move away from using "namespace/name" format [https://github.com/openservicemesh/osm/issues/2218]
	// Currently this will be: "service-cert:default/bookbuyer"
	return fmt.Sprintf("service-cert:%s", si.ToK8sServiceAccount())
}

// NameForUpstreamService returns the SDS secret name corresponding to the given outbound Service name and namespace.
func NameForUpstreamService(name, namespace string) string {
	return fmt.Sprintf("root-cert-for-mtls-outbound:%s/%s", namespace, name)
}
