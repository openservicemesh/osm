package catalog

import (
	"fmt"

	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/trafficpolicy"
)

// GetIngressPoliciesForService returns a list of inbound traffic policies for a service as defined in observed ingress k8s resources.
func (mc *MeshCatalog) GetIngressPoliciesForService(svc service.MeshService) ([]*trafficpolicy.InboundTrafficPolicy, error) {
	inboundIngressPolicies := []*trafficpolicy.InboundTrafficPolicy{}
	// Ingress does not depend on k8s service accounts, program a wildcard (empty struct) to indicate
	// to RDS that an inbound traffic policy for ingress should not enforce service account based RBAC policies.
	wildcardServiceAccount := service.K8sServiceAccount{}

	ingresses, err := mc.ingressMonitor.GetIngressResources(svc)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to get ingress resources for service %s", svc)
		return inboundIngressPolicies, err
	}
	if len(ingresses) == 0 {
		log.Trace().Msgf("No ingress resources found for service %s", svc)
		return inboundIngressPolicies, err
	}

	ingressWeightedCluster := getDefaultWeightedClusterForService(svc)

	for _, ingress := range ingresses {
		if ingress.Spec.Backend != nil && ingress.Spec.Backend.ServiceName == svc.Name {
			wildcardIngressPolicy := trafficpolicy.NewInboundTrafficPolicy(buildIngressPolicyName(ingress.ObjectMeta.Name, ingress.ObjectMeta.Namespace, constants.WildcardHTTPMethod), []string{constants.WildcardHTTPMethod})
			wildcardIngressPolicy.AddRule(*trafficpolicy.NewRouteWeightedCluster(wildCardRouteMatch, ingressWeightedCluster), wildcardServiceAccount)
			inboundIngressPolicies = trafficpolicy.MergeInboundPolicies(false, inboundIngressPolicies, wildcardIngressPolicy)
		}

		for _, rule := range ingress.Spec.Rules {
			domain := rule.Host
			if domain == "" {
				domain = constants.WildcardHTTPMethod
			}
			ingressPolicy := trafficpolicy.NewInboundTrafficPolicy(buildIngressPolicyName(ingress.ObjectMeta.Name, ingress.ObjectMeta.Namespace, domain), []string{domain})

			for _, ingressPath := range rule.HTTP.Paths {
				if ingressPath.Backend.ServiceName != svc.Name {
					continue
				}
				routePolicy := wildCardRouteMatch
				if routePolicy.PathRegex != "" {
					routePolicy.PathRegex = ingressPath.Path
				}
				ingressPolicy.AddRule(*trafficpolicy.NewRouteWeightedCluster(routePolicy, ingressWeightedCluster), wildcardServiceAccount)
			}

			inboundIngressPolicies = trafficpolicy.MergeInboundPolicies(false, inboundIngressPolicies, ingressPolicy)
		}
	}
	return inboundIngressPolicies, nil
}

func buildIngressPolicyName(name, namespace, host string) string {
	policyName := fmt.Sprintf("%s.%s|%s", name, namespace, host)
	return policyName
}
