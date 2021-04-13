package catalog

import (
	"fmt"

	access "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/access/v1alpha3"

	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/trafficpolicy"
)

const (
	// AllowPartialHostnamesMatch is used to allow a partial/subset match on hostnames in traffic policies
	AllowPartialHostnamesMatch bool = true
	// DisallowPartialHostnamesMatch is used to disallow a partial/subset match on hostnames in traffic policies
	DisallowPartialHostnamesMatch bool = false
)

// ListInboundTrafficPolicies returns all inbound traffic policies
// 1. from service discovery for permissive mode
// 2. for the given service account and upstream services from SMI Traffic Target and Traffic Split
func (mc *MeshCatalog) ListInboundTrafficPolicies(upstreamIdentity service.K8sServiceAccount, upstreamServices []service.MeshService) []*trafficpolicy.InboundTrafficPolicy {
	if mc.configurator.IsPermissiveTrafficPolicyMode() {
		inboundPolicies := []*trafficpolicy.InboundTrafficPolicy{}
		for _, svc := range upstreamServices {
			inboundPolicies = trafficpolicy.MergeInboundPolicies(DisallowPartialHostnamesMatch, inboundPolicies, mc.buildInboundPermissiveModePolicies(svc)...)
		}
		return inboundPolicies
	}

	inbound := mc.listInboundPoliciesFromTrafficTargets(upstreamIdentity, upstreamServices)
	inboundPoliciesFRomSplits := mc.listInboundPoliciesForTrafficSplits(upstreamIdentity, upstreamServices)
	inbound = trafficpolicy.MergeInboundPolicies(DisallowPartialHostnamesMatch, inbound, inboundPoliciesFRomSplits...)
	return inbound
}

// listInboundPoliciesFromTrafficTargets builds inbound traffic policies for all inbound services
// when the given service account matches a destination in the Traffic Target resource
func (mc *MeshCatalog) listInboundPoliciesFromTrafficTargets(upstreamIdentity service.K8sServiceAccount, upstreamServices []service.MeshService) []*trafficpolicy.InboundTrafficPolicy {
	inboundPolicies := []*trafficpolicy.InboundTrafficPolicy{}

	for _, t := range mc.meshSpec.ListTrafficTargets() { // loop through all traffic targets
		if !isValidTrafficTarget(t) {
			continue
		}

		if t.Spec.Destination.Name != upstreamIdentity.Name { // not an inbound policy for the upstream services
			continue
		}

		for _, svc := range upstreamServices {
			inboundPolicies = trafficpolicy.MergeInboundPolicies(DisallowPartialHostnamesMatch, inboundPolicies, mc.buildInboundPolicies(t, svc)...)
		}
	}

	return inboundPolicies
}

// listInboundPoliciesForTrafficSplits loops through all SMI TrafficTarget resources and returns inbound policies for apex services based on the following conditions:
// 1. the given upstream identity matches the destination specified in a TrafficTarget resource
// 2. the given list of upstream services are backends specified in a TrafficSplit resource
func (mc *MeshCatalog) listInboundPoliciesForTrafficSplits(upstreamIdentity service.K8sServiceAccount, upstreamServices []service.MeshService) []*trafficpolicy.InboundTrafficPolicy {
	inboundPolicies := []*trafficpolicy.InboundTrafficPolicy{}

	for _, t := range mc.meshSpec.ListTrafficTargets() { // loop through all traffic targets
		if !isValidTrafficTarget(t) {
			continue
		}

		if t.Spec.Destination.Name != upstreamIdentity.Name { // not an inbound policy for the upstream identity
			continue
		}

		// fetch all routes referenced in traffic target
		routeMatches, err := mc.routesFromRules(t.Spec.Rules, t.Namespace)
		if err != nil {
			log.Error().Err(err).Msgf("Error finding route matches from TrafficTarget %s in namespace %s", t.Name, t.Namespace)
			continue
		}

		for _, upstreamSvc := range upstreamServices {
			//check if the upstream service belong to a traffic split
			if !mc.isTrafficSplitBackendService(upstreamSvc) {
				continue
			}

			apexServices := mc.getApexServicesForBackendService(upstreamSvc)
			for _, apexService := range apexServices {
				// build an inbound policy for every apex service
				hostnames, err := mc.getServiceHostnames(apexService, apexService.Namespace == upstreamIdentity.Namespace)
				if err != nil {
					log.Error().Err(err).Msgf("Error getting service hostnames for apex service %v", apexService)
					continue
				}
				servicePolicy := trafficpolicy.NewInboundTrafficPolicy(buildPolicyName(apexService, apexService.Namespace == upstreamIdentity.Namespace), hostnames)
				weightedCluster := getDefaultWeightedClusterForService(upstreamSvc)

				for _, sourceServiceAccount := range trafficTargetIdentitiesToSvcAccounts(t.Spec.Sources) {
					for _, routeMatch := range routeMatches {
						servicePolicy.AddRule(*trafficpolicy.NewRouteWeightedCluster(routeMatch, []service.WeightedCluster{weightedCluster}), sourceServiceAccount)
					}
				}
				inboundPolicies = trafficpolicy.MergeInboundPolicies(DisallowPartialHostnamesMatch, inboundPolicies, servicePolicy)
			}
		}
	}
	return inboundPolicies
}

func (mc *MeshCatalog) buildInboundPolicies(t *access.TrafficTarget, svc service.MeshService) []*trafficpolicy.InboundTrafficPolicy {
	inboundPolicies := []*trafficpolicy.InboundTrafficPolicy{}

	// fetch all routes referenced in traffic target
	routeMatches, err := mc.routesFromRules(t.Spec.Rules, t.Namespace)
	if err != nil {
		log.Error().Err(err).Msgf("Error finding route matches from TrafficTarget %s in namespace %s", t.Name, t.Namespace)
		return inboundPolicies
	}

	hostnames, err := mc.getServiceHostnames(svc, true)
	if err != nil {
		log.Error().Err(err).Msgf("Error getting service hostnames for service %s", svc)
		return inboundPolicies
	}

	servicePolicy := trafficpolicy.NewInboundTrafficPolicy(buildPolicyName(svc, false), hostnames)
	weightedCluster := getDefaultWeightedClusterForService(svc)

	for _, sourceServiceAccount := range trafficTargetIdentitiesToSvcAccounts(t.Spec.Sources) {
		for _, routeMatch := range routeMatches {
			servicePolicy.AddRule(*trafficpolicy.NewRouteWeightedCluster(routeMatch, []service.WeightedCluster{weightedCluster}), sourceServiceAccount)
		}
	}

	if len(servicePolicy.Rules) > 0 {
		inboundPolicies = append(inboundPolicies, servicePolicy)
	}

	return inboundPolicies
}

func (mc *MeshCatalog) buildInboundPermissiveModePolicies(svc service.MeshService) []*trafficpolicy.InboundTrafficPolicy {
	inboundPolicies := []*trafficpolicy.InboundTrafficPolicy{}

	hostnames, err := mc.getServiceHostnames(svc, true)
	if err != nil {
		log.Error().Err(err).Msgf("Error getting service hostnames for service %s", svc)
		return inboundPolicies
	}

	servicePolicy := trafficpolicy.NewInboundTrafficPolicy(buildPolicyName(svc, false), hostnames)
	weightedCluster := getDefaultWeightedClusterForService(svc)

	// Add a wildcard route to accept traffic from any service account (wildcard service account)
	// A wildcard service account will program an RBAC policy for this rule that allows ANY downstream service account
	servicePolicy.AddRule(*trafficpolicy.NewRouteWeightedCluster(trafficpolicy.WildCardRouteMatch, []service.WeightedCluster{weightedCluster}), wildcardServiceAccount)
	inboundPolicies = append(inboundPolicies, servicePolicy)

	return inboundPolicies
}

// routesFromRules takes a set of traffic target rules and the namespace of the traffic target and returns a list of
//	http route matches (trafficpolicy.HTTPRouteMatch)
func (mc *MeshCatalog) routesFromRules(rules []access.TrafficTargetRule, trafficTargetNamespace string) ([]trafficpolicy.HTTPRouteMatch, error) {
	routes := []trafficpolicy.HTTPRouteMatch{}

	specMatchRoute, err := mc.getHTTPPathsPerRoute() // returns map[traffic_spec_name]map[match_name]trafficpolicy.HTTPRoute
	if err != nil {
		return nil, err
	}

	if len(specMatchRoute) == 0 {
		log.Trace().Msg("No elements in map[traffic_spec_name]map[match name]trafficpolicyHTTPRoute")
		return routes, nil
	}

	for _, rule := range rules {
		trafficSpecName := mc.getTrafficSpecName("HTTPRouteGroup", trafficTargetNamespace, rule.Name)
		for _, match := range rule.Matches {
			matchedRoute, found := specMatchRoute[trafficSpecName][trafficpolicy.TrafficSpecMatchName(match)]
			if found {
				routes = append(routes, matchedRoute)
			} else {
				log.Debug().Msgf("No matching trafficpolicy.HTTPRoute found for match name %s in Traffic Spec %s (in namespace %s)", match, trafficSpecName, trafficTargetNamespace)
			}
		}
	}

	return routes, nil
}

func (mc *MeshCatalog) getHTTPPathsPerRoute() (map[trafficpolicy.TrafficSpecName]map[trafficpolicy.TrafficSpecMatchName]trafficpolicy.HTTPRouteMatch, error) {
	routePolicies := make(map[trafficpolicy.TrafficSpecName]map[trafficpolicy.TrafficSpecMatchName]trafficpolicy.HTTPRouteMatch)
	for _, trafficSpecs := range mc.meshSpec.ListHTTPTrafficSpecs() {
		log.Debug().Msgf("Discovered TrafficSpec resource: %s/%s", trafficSpecs.Namespace, trafficSpecs.Name)
		if trafficSpecs.Spec.Matches == nil {
			log.Error().Msgf("TrafficSpec %s/%s has no matches in route; Skipping...", trafficSpecs.Namespace, trafficSpecs.Name)
			continue
		}

		// since this method gets only specs related to HTTPRouteGroups added HTTPTraffic to the specKey by default
		specKey := mc.getTrafficSpecName(httpRouteGroupKind, trafficSpecs.Namespace, trafficSpecs.Name)
		routePolicies[specKey] = make(map[trafficpolicy.TrafficSpecMatchName]trafficpolicy.HTTPRouteMatch)
		for _, trafficSpecsMatches := range trafficSpecs.Spec.Matches {
			serviceRoute := trafficpolicy.HTTPRouteMatch{
				Path:          trafficSpecsMatches.PathRegex,
				PathMatchType: trafficpolicy.PathMatchRegex,
				Methods:       trafficSpecsMatches.Methods,
				Headers:       trafficSpecsMatches.Headers,
			}

			// When pathRegex or/and methods are not defined, they will be wildcarded
			if serviceRoute.Path == "" {
				serviceRoute.Path = constants.RegexMatchAll
			}
			if len(serviceRoute.Methods) == 0 {
				serviceRoute.Methods = []string{constants.WildcardHTTPMethod}
			}
			routePolicies[specKey][trafficpolicy.TrafficSpecMatchName(trafficSpecsMatches.Name)] = serviceRoute
		}
	}
	log.Debug().Msgf("Constructed HTTP path routes: %+v", routePolicies)
	return routePolicies, nil
}

func (mc *MeshCatalog) getTrafficSpecName(trafficSpecKind string, trafficSpecNamespace string, trafficSpecName string) trafficpolicy.TrafficSpecName {
	specKey := fmt.Sprintf("%s/%s/%s", trafficSpecKind, trafficSpecNamespace, trafficSpecName)
	return trafficpolicy.TrafficSpecName(specKey)
}

// buildPolicyName creates a name for a policy associated with the given service
func buildPolicyName(svc service.MeshService, sameNamespace bool) string {
	name := svc.Name
	if !sameNamespace {
		return name + "." + svc.Namespace
	}
	return name
}
