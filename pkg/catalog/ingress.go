package catalog

import (
	"fmt"

	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/trafficpolicy"
)

// GetIngressRoutesPerHost returns route matches per host as defined in observed ingress k8s resources.
// TODO : Remove as a part of routes refactor cleanup (#2397)
func (mc *MeshCatalog) GetIngressRoutesPerHost(service service.MeshService) (map[string][]trafficpolicy.HTTPRouteMatch, error) {
	domainRoutesMap := make(map[string][]trafficpolicy.HTTPRouteMatch)
	ingresses, err := mc.ingressMonitor.GetIngressResources(service)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to get ingress resources with backend %s", service)
		return domainRoutesMap, err
	}
	if len(ingresses) == 0 {
		return domainRoutesMap, err
	}

	defaultRoute := trafficpolicy.HTTPRouteMatch{
		PathRegex: constants.RegexMatchAll,
		Methods:   []string{constants.RegexMatchAll},
	}

	for _, ingress := range ingresses {
		if ingress.Spec.Backend != nil && ingress.Spec.Backend.ServiceName == service.Name {
			domainRoutesMap[constants.WildcardHTTPMethod] = []trafficpolicy.HTTPRouteMatch{defaultRoute}
		}

		for _, rule := range ingress.Spec.Rules {
			domain := rule.Host
			if domain == "" {
				domain = constants.WildcardHTTPMethod
			}
			for _, ingressPath := range rule.HTTP.Paths {
				if ingressPath.Backend.ServiceName != service.Name {
					continue
				}
				routePolicy := defaultRoute
				if routePolicy.PathRegex != "" {
					routePolicy.PathRegex = ingressPath.Path
				}
				domainRoutesMap[domain] = append(domainRoutesMap[domain], routePolicy)
			}
		}
	}

	log.Trace().Msgf("Created routes per host for service %s: %+v", service, domainRoutesMap)

	return domainRoutesMap, nil
}

// GetIngressPoliciesForService returns a list of inbound traffic policies for a service as defined in observed ingress k8s resources.
func (mc *MeshCatalog) GetIngressPoliciesForService(svc service.MeshService, sa service.K8sServiceAccount) ([]*trafficpolicy.InboundTrafficPolicy, error) {
	inboundIngressPolicies := []*trafficpolicy.InboundTrafficPolicy{}
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
			wildcardIngressPolicy.AddRule(*trafficpolicy.NewRouteWeightedCluster(wildCardRouteMatch, ingressWeightedCluster), sa)
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
				ingressPolicy.AddRule(*trafficpolicy.NewRouteWeightedCluster(routePolicy, ingressWeightedCluster), sa)
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
