package trafficpolicy

import (
	"fmt"

	"k8s.io/apimachinery/pkg/types"
)

// IngressTrafficPolicy defines the ingress traffic match and routes for a given backend
type IngressTrafficPolicy struct {
	TrafficMatches    []*IngressTrafficMatch
	HTTPRoutePolicies []*InboundTrafficPolicy
}

// IngressTrafficMatch defines the attributes to match ingress traffic for a given backend
type IngressTrafficMatch struct {
	Name                     string
	Port                     uint32
	Protocol                 string
	SourceIPRanges           []string
	ServerNames              []string
	SkipClientCertValidation bool
}

// GetIngressTrafficMatchName generates the traffic match name
func GetIngressTrafficMatchName(svc types.NamespacedName, port uint16, protocol string) string {
	return fmt.Sprintf("ingress_%s/%s_%d_%s", svc.Namespace, svc.Name, port, protocol)
}
