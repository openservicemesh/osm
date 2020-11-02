package identity

import (
	"strings"

	"github.com/openservicemesh/osm/pkg/service"
)

const (
	identityDelimiter = "."
)

// GetKubernetesServiceIdentity returns the ServiceIdentity based on Kubernetes ServiceAccount and a trust domain
func GetKubernetesServiceIdentity(svcAccount service.K8sServiceAccount, trustDomain string) ServiceIdentity {
	si := strings.Join([]string{svcAccount.Name, svcAccount.Namespace, trustDomain}, identityDelimiter)
	return ServiceIdentity(si)
}
