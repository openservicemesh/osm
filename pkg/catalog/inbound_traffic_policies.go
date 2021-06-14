package catalog

import (
	"fmt"

	access "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/access/v1alpha3"

	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/trafficpolicy"
)

const (
	// AllowPartialHostnamesMatch is used to allow a partial/subset match on hostnames in traffic policies
	AllowPartialHostnamesMatch bool = true
	// DisallowPartialHostnamesMatch is used to disallow a partial/subset match on hostnames in traffic policies
	DisallowPartialHostnamesMatch bool = false
	// hostHeaderHey is a string to represent the header key 'host'
	hostHeaderKey string = "host"
)

// ListInboundTrafficPolicies returns all inbound traffic policies
// 1. from service discovery for permissive mode
// 2. for the given service account and upstream services from SMI Traffic Target and Traffic Split
// Note: ServiceIdentity must be in the format "name.namespace" [https://github.com/openservicemesh/osm/issues/3188]
func (mc *MeshCatalog) ListInboundTrafficPolicies(upstreamIdentity identity.ServiceIdentity, upstreamServices []service.MeshService) []*trafficpolicy.InboundTrafficPolicy {
	if mc.configurator.IsPermissiveTrafficPolicyMode() {
		var inboundPolicies []*trafficpolicy.InboundTrafficPolicy
		for _, svc := range upstreamServices {
			inboundPolicies = trafficpolicy.MergeInboundPolicies(DisallowPartialHostnamesMatch, inboundPolicies, mc.buildInboundPermissiveModePolicies(svc)...)
		}
		return inboundPolicies
	}

	inbound := mc.listInboundPoliciesFromTrafficTargets(upstreamIdentity, upstreamServices)
	inboundPoliciesFromSplits := mc.listInboundPoliciesForTrafficSplits(upstreamIdentity, upstreamServices)
	inbound = trafficpolicy.MergeInboundPolicies(AllowPartialHostnamesMatch, inbound, inboundPoliciesFromSplits...)
	return inbound
}

// listInboundPoliciesFromTrafficTargets builds inbound traffic policies for all inbound services
// when the given service account matches a destination in the Traffic Target resource
// Note: ServiceIdentity must be in the format "name.namespace" [https://github.com/openservicemesh/osm/issues/3188]
func (mc *MeshCatalog) listInboundPoliciesFromTrafficTargets(upstreamIdentity identity.ServiceIdentity, upstreamServices []service.MeshService) []*trafficpolicy.InboundTrafficPolicy {
	upstreamServiceAccount := upstreamIdentity.ToK8sServiceAccount()
	var inboundPolicies []*trafficpolicy.InboundTrafficPolicy

	for _, t := range mc.meshSpec.ListTrafficTargets() { // loop through all traffic targets
		if !isValidTrafficTarget(t) {
			continue
		}

		// TODO(draychev): Add a check to ensure that ServiceIdentities are of the same kind! [https://github.com/openservicemesh/osm/issues/3173]
		if t.Spec.Destination.Name != upstreamServiceAccount.Name { // not an inbound policy for the upstream services
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
// Note: ServiceIdentity must be in the format "name.namespace" [https://github.com/openservicemesh/osm/issues/3188]
func (mc *MeshCatalog) listInboundPoliciesForTrafficSplits(upstreamIdentity identity.ServiceIdentity, upstreamServices []service.MeshService) []*trafficpolicy.InboundTrafficPolicy {
	upstreamServiceAccount := upstreamIdentity.ToK8sServiceAccount()
	var inboundPolicies []*trafficpolicy.InboundTrafficPolicy

	for _, t := range mc.meshSpec.ListTrafficTargets() { // loop through all traffic targets
		if !isValidTrafficTarget(t) {
			continue
		}

		// TODO(draychev): Add a check to ensure that ServiceIdentities are of the same kind! [https://github.com/openservicemesh/osm/issues/3173]
		if t.Spec.Destination.Name != upstreamServiceAccount.Name { // not an inbound policy for the upstream identity
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
				locality := service.LocalCluster
				if apexService.Namespace == upstreamServiceAccount.Namespace {
					locality = service.LocalNS
				}
				hostnames, err := mc.GetServiceHostnames(apexService, locality)
				if err != nil {
					log.Error().Err(err).Msgf("Error getting service hostnames for apex service %v", apexService)
					continue
				}
				servicePolicy := trafficpolicy.NewInboundTrafficPolicy(apexService.FQDN(), hostnames)
				weightedCluster := getDefaultWeightedClusterForService(upstreamSvc)

				for _, sourceServiceAccount := range trafficTargetIdentitiesToSvcAccounts(t.Spec.Sources) {
					for _, routeMatch := range routeMatches {
						// If the traffic target has a route with host headers
						// we need to create a new inbound traffic policy with the host header as the required hostnames
						// else the hosnames will be hostnames corresponding to the service
						if _, ok := routeMatch.Headers[hostHeaderKey]; !ok {
							servicePolicy.AddRule(*trafficpolicy.NewRouteWeightedCluster(routeMatch, []service.WeightedCluster{weightedCluster}), sourceServiceAccount)
						} else {
							servicePolicyWithHostHeader := trafficpolicy.NewInboundTrafficPolicy(routeMatch.Headers[hostHeaderKey], []string{routeMatch.Headers[hostHeaderKey]})
							servicePolicyWithHostHeader.AddRule(*trafficpolicy.NewRouteWeightedCluster(routeMatch, []service.WeightedCluster{weightedCluster}), sourceServiceAccount)
							inboundPolicies = trafficpolicy.MergeInboundPolicies(AllowPartialHostnamesMatch, inboundPolicies, servicePolicyWithHostHeader)
						}
					}
				}
				inboundPolicies = trafficpolicy.MergeInboundPolicies(AllowPartialHostnamesMatch, inboundPolicies, servicePolicy)
			}
		}
	}
	return inboundPolicies
}

func (mc *MeshCatalog) buildInboundPolicies(t *access.TrafficTarget, svc service.MeshService) []*trafficpolicy.InboundTrafficPolicy {
	var inboundPolicies []*trafficpolicy.InboundTrafficPolicy

	// fetch all routes referenced in traffic target
	routeMatches, err := mc.routesFromRules(t.Spec.Rules, t.Namespace)
	if err != nil {
		log.Error().Err(err).Msgf("Error finding route matches from TrafficTarget %s in namespace %s", t.Name, t.Namespace)
		return inboundPolicies
	}

	hostnames, err := mc.GetServiceHostnames(svc, service.LocalNS)
	if err != nil {
		log.Error().Err(err).Msgf("Error getting service hostnames for service %s", svc)
		return inboundPolicies
	}

	servicePolicy := trafficpolicy.NewInboundTrafficPolicy(svc.FQDN(), hostnames)
	weightedCluster := getDefaultWeightedClusterForService(svc)

	for _, sourceServiceAccount := range trafficTargetIdentitiesToSvcAccounts(t.Spec.Sources) {
		for _, routeMatch := range routeMatches {
			// If the traffic target has a route with host headers
			// we need to create a new inbound traffic policy with the host header as the required hostnames
			// else the hosnames will be hostnames corresponding to the service
			if _, ok := routeMatch.Headers[hostHeaderKey]; !ok {
				servicePolicy.AddRule(*trafficpolicy.NewRouteWeightedCluster(routeMatch, []service.WeightedCluster{weightedCluster}), sourceServiceAccount)
			} else {
				servicePolicyWithHostHeader := trafficpolicy.NewInboundTrafficPolicy(routeMatch.Headers[hostHeaderKey], []string{routeMatch.Headers[hostHeaderKey]})
				servicePolicyWithHostHeader.AddRule(*trafficpolicy.NewRouteWeightedCluster(routeMatch, []service.WeightedCluster{weightedCluster}), sourceServiceAccount)
				inboundPolicies = trafficpolicy.MergeInboundPolicies(AllowPartialHostnamesMatch, inboundPolicies, servicePolicyWithHostHeader)
			}
		}
	}

	inboundPolicies = trafficpolicy.MergeInboundPolicies(AllowPartialHostnamesMatch, inboundPolicies, servicePolicy)

	return inboundPolicies
}

func (mc *MeshCatalog) buildInboundPermissiveModePolicies(svc service.MeshService) []*trafficpolicy.InboundTrafficPolicy {
	var inboundPolicies []*trafficpolicy.InboundTrafficPolicy

	hostnames, err := mc.GetServiceHostnames(svc, service.LocalNS)
	if err != nil {
		log.Error().Err(err).Msgf("Error getting service hostnames for service %s", svc)
		return inboundPolicies
	}

	servicePolicy := trafficpolicy.NewInboundTrafficPolicy(svc.FQDN(), hostnames)
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
	var routes []trafficpolicy.HTTPRouteMatch

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
