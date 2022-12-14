package catalog

import (
	"fmt"

	mapset "github.com/deckarep/golang-set"
	access "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/access/v1alpha3"

	policyv1alpha1 "github.com/openservicemesh/osm/pkg/apis/policy/v1alpha1"

	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/errcode"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/k8s"
	"github.com/openservicemesh/osm/pkg/policy"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/smi"
	"github.com/openservicemesh/osm/pkg/trafficpolicy"
)

const (
	// AllowPartialHostnamesMatch is used to allow a partial/subset match on hostnames in traffic policies
	AllowPartialHostnamesMatch bool = true

	// DisallowPartialHostnamesMatch is used to disallow a partial/subset match on hostnames in traffic policies
	DisallowPartialHostnamesMatch bool = false
)

// GetInboundMeshTrafficPolicy returns the inbound mesh traffic policy for the given upstream identity and services
func (mc *MeshCatalog) GetInboundMeshTrafficPolicy(upstreamIdentity identity.ServiceIdentity, upstreamServices []service.MeshService) *trafficpolicy.InboundMeshTrafficPolicy {
	var trafficMatches []*trafficpolicy.TrafficMatch
	var clusterConfigs []*trafficpolicy.MeshClusterConfig
	var trafficTargets []*access.TrafficTarget
	routeConfigPerPort := make(map[int][]*trafficpolicy.InboundTrafficPolicy)

	permissiveMode := mc.configurator.IsPermissiveTrafficPolicyMode()
	if !permissiveMode {
		// Pre-computing the list of TrafficTarget optimizes to avoid repeated
		// cache lookups for each upstream service.
		destinationFilter := smi.WithTrafficTargetDestination(upstreamIdentity.ToK8sServiceAccount())
		trafficTargets = mc.meshSpec.ListTrafficTargets(destinationFilter)
	}

	upstreamSvcSet := mapset.NewSet()
	for _, svc := range upstreamServices {
		upstreamSvcSet.Add(svc)
	}

	// A policy (traffic match, route, cluster) must be built for each upstream service. This
	// includes apex/root services associated with the given upstream service.
	allUpstreamServices := mc.getUpstreamServicesIncludeApex(upstreamServices)

	// Build configurations per upstream service
	for _, upstreamSvc := range allUpstreamServices {
		upstreamSvc := upstreamSvc // To prevent loop variable memory aliasing in for loop

		// ---
		// Create local cluster configs for this upstram service
		clusterConfigForSvc := &trafficpolicy.MeshClusterConfig{
			Name:    upstreamSvc.EnvoyLocalClusterName(),
			Service: upstreamSvc,
			Address: constants.LocalhostIPAddress,
			Port:    uint32(upstreamSvc.TargetPort),
		}
		clusterConfigs = append(clusterConfigs, clusterConfigForSvc)

		upstreamTrafficSetting := mc.policyController.GetUpstreamTrafficSetting(
			policy.UpstreamTrafficSettingGetOpt{MeshService: &upstreamSvc})

		// ---
		// Create a TrafficMatch for this upstream servic.
		// The TrafficMatch will be used by LDS to program a filter chain match
		// for this upstream service on the upstream server to accept inbound
		// traffic.
		//
		// Note: a TrafficMatch must exist only for a service part of the given
		// 'upstreamServices' list, and not a virtual (apex) service that
		// may be returned as a part of the 'allUpstreamServices' list.
		// A virtual (apex) service is required for the purpose of building
		// HTTP routing rules, but should not result in a TrafficMatch rule
		// as TrafficMatch rules are meant to map to actual services backed
		// by a proxy, defined by the 'upstreamServices' list.
		if upstreamSvcSet.Contains(upstreamSvc) {
			trafficMatchForUpstreamSvc := &trafficpolicy.TrafficMatch{
				Name:                upstreamSvc.InboundTrafficMatchName(),
				DestinationPort:     int(upstreamSvc.TargetPort),
				DestinationProtocol: upstreamSvc.Protocol,
				ServerNames:         []string{upstreamSvc.ServerName()},
				Cluster:             upstreamSvc.EnvoyLocalClusterName(),
			}
			if upstreamTrafficSetting != nil {
				trafficMatchForUpstreamSvc.RateLimit = upstreamTrafficSetting.Spec.RateLimit
			}
			trafficMatches = append(trafficMatches, trafficMatchForUpstreamSvc)
		}

		// Build the HTTP route configs for this service and port combination.
		// If the port's protocol corresponds to TCP, we can skip this step
		if upstreamSvc.Protocol == constants.ProtocolTCP || upstreamSvc.Protocol == constants.ProtocolTCPServerFirst {
			continue
		}
		// ---
		// Build the HTTP route configs per port
		// Each upstream service accepts traffic from downstreams on a list of allowed routes.
		// The routes are derived from SMI TrafficTarget and TrafficSplit policies in SMI mode,
		// and are wildcarded in permissive mode. The downstreams that can access this upstream
		// on the configured routes is also determined based on the traffic policy mode.
		inboundTrafficPolicies := mc.getInboundTrafficPoliciesForUpstream(upstreamSvc, permissiveMode, trafficTargets, upstreamTrafficSetting)
		routeConfigPerPort[int(upstreamSvc.TargetPort)] = append(routeConfigPerPort[int(upstreamSvc.TargetPort)], inboundTrafficPolicies)
	}

	return &trafficpolicy.InboundMeshTrafficPolicy{
		TrafficMatches:          trafficMatches,
		ClustersConfigs:         clusterConfigs,
		HTTPRouteConfigsPerPort: routeConfigPerPort,
	}
}

func (mc *MeshCatalog) getInboundTrafficPoliciesForUpstream(upstreamSvc service.MeshService, permissiveMode bool,
	trafficTargets []*access.TrafficTarget, upstreamTrafficSetting *policyv1alpha1.UpstreamTrafficSetting) *trafficpolicy.InboundTrafficPolicy {
	var inboundPolicyForUpstreamSvc *trafficpolicy.InboundTrafficPolicy

	if permissiveMode {
		// Add a wildcard HTTP route that allows any downstream client to access the upstream service
		hostnames := k8s.GetHostnamesForService(upstreamSvc, true /* local namespace FQDN should always be allowed for inbound routes*/)
		inboundPolicyForUpstreamSvc = trafficpolicy.NewInboundTrafficPolicy(upstreamSvc.FQDN(), hostnames, upstreamTrafficSetting)
		localCluster := service.WeightedCluster{
			ClusterName: service.ClusterName(upstreamSvc.EnvoyLocalClusterName()),
			Weight:      constants.ClusterWeightAcceptAll,
		}
		// Only a single rule for permissive mode.
		inboundPolicyForUpstreamSvc.Rules = []*trafficpolicy.Rule{
			{
				Route:             *trafficpolicy.NewRouteWeightedCluster(trafficpolicy.WildCardRouteMatch, []service.WeightedCluster{localCluster}, upstreamTrafficSetting),
				AllowedPrincipals: mapset.NewSetWith(identity.WildcardPrincipal),
			},
		}
	} else {
		// Build the HTTP routes from SMI TrafficTarget and HTTPRouteGroup configurations
		inboundPolicyForUpstreamSvc = mc.buildInboundHTTPPolicyFromTrafficTarget(upstreamSvc, trafficTargets, upstreamTrafficSetting)
	}

	return inboundPolicyForUpstreamSvc
}

func (mc *MeshCatalog) buildInboundHTTPPolicyFromTrafficTarget(upstreamSvc service.MeshService, trafficTargets []*access.TrafficTarget,
	upstreamTrafficSetting *policyv1alpha1.UpstreamTrafficSetting) *trafficpolicy.InboundTrafficPolicy {
	hostnames := k8s.GetHostnamesForService(upstreamSvc, true /* local namespace FQDN should always be allowed for inbound routes*/)
	inboundPolicy := trafficpolicy.NewInboundTrafficPolicy(upstreamSvc.FQDN(), hostnames, upstreamTrafficSetting)

	localCluster := service.WeightedCluster{
		ClusterName: service.ClusterName(upstreamSvc.EnvoyLocalClusterName()),
		Weight:      constants.ClusterWeightAcceptAll,
	}

	var routingRules []*trafficpolicy.Rule
	// From each TrafficTarget and HTTPRouteGroup configuration associated with this service, build routes for it.
	for _, trafficTarget := range trafficTargets {
		rules := mc.getRoutingRulesFromTrafficTarget(*trafficTarget, localCluster, upstreamTrafficSetting)
		// Multiple TrafficTarget objects can reference the same route, in which case such routes
		// need to be merged to create a single route that includes all the downstream client identities
		// this route is authorized for.
		routingRules = trafficpolicy.MergeRules(routingRules, rules)
	}
	inboundPolicy.Rules = routingRules

	return inboundPolicy
}

func (mc *MeshCatalog) getRoutingRulesFromTrafficTarget(trafficTarget access.TrafficTarget, routingCluster service.WeightedCluster,
	upstreamTrafficSetting *policyv1alpha1.UpstreamTrafficSetting) []*trafficpolicy.Rule {
	// Compute the HTTP route matches associated with the given TrafficTarget object
	httpRouteMatches, err := mc.routesFromRules(trafficTarget.Spec.Rules, trafficTarget.Namespace)
	if err != nil {
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrFetchingSMIHTTPRouteGroupForTrafficTarget)).
			Msgf("Error finding route matches from TrafficTarget %s in namespace %s", trafficTarget.Name, trafficTarget.Namespace)
		return nil
	}

	// Compute the allowed downstream service identities for the given TrafficTarget object
	trustDomain := mc.GetTrustDomain()
	allowedDownstreamPrincipals := mapset.NewSet()
	for _, source := range trafficTarget.Spec.Sources {
		allowedDownstreamPrincipals.Add(trafficTargetIdentityToSvcAccount(source).AsPrincipal(trustDomain))
	}

	var routingRules []*trafficpolicy.Rule
	for _, httpRouteMatch := range httpRouteMatches {
		rule := &trafficpolicy.Rule{
			Route:             *trafficpolicy.NewRouteWeightedCluster(httpRouteMatch, []service.WeightedCluster{routingCluster}, upstreamTrafficSetting),
			AllowedPrincipals: allowedDownstreamPrincipals,
		}
		routingRules = append(routingRules, rule)
	}

	return routingRules
}

// routesFromRules takes a set of traffic target rules and the namespace of the traffic target and returns a list of
//
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
		trafficSpecName := getTrafficSpecName(smi.HTTPRouteGroupKind, trafficTargetNamespace, rule.Name)
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
			log.Error().Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrSMIHTTPRouteGroupNoMatch)).
				Msgf("TrafficSpec %s/%s has no matches in route; Skipping...", trafficSpecs.Namespace, trafficSpecs.Name)
			continue
		}

		// since this method gets only specs related to HTTPRouteGroups added HTTPTraffic to the specKey by default
		specKey := getTrafficSpecName(smi.HTTPRouteGroupKind, trafficSpecs.Namespace, trafficSpecs.Name)
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

func getTrafficSpecName(trafficSpecKind string, trafficSpecNamespace string, trafficSpecName string) trafficpolicy.TrafficSpecName {
	specKey := fmt.Sprintf("%s/%s/%s", trafficSpecKind, trafficSpecNamespace, trafficSpecName)
	return trafficpolicy.TrafficSpecName(specKey)
}

// getUpstreamServicesIncludeApex returns a list of all upstream services associated with the given list
// of services. An upstream service is associated with another service if it is a backend for an apex/root service
// in a TrafficSplit config. This function returns a list consisting of the given upstream services and all apex
// services associated with each of those services.
func (mc *MeshCatalog) getUpstreamServicesIncludeApex(upstreamServices []service.MeshService) []service.MeshService {
	svcSet := mapset.NewSet()
	var allServices []service.MeshService

	// Each service could be a backend in a traffic split config. Construct a list
	// of all possible services the given list of services is associated with.
	for _, svc := range upstreamServices {
		if newlyAdded := svcSet.Add(svc); newlyAdded {
			allServices = append(allServices, svc)
		}

		for _, split := range mc.meshSpec.ListTrafficSplits(smi.WithTrafficSplitBackendService(svc)) {
			svcName := k8s.GetServiceFromHostname(mc.kubeController, split.Spec.Service)
			subdomain := k8s.GetSubdomainFromHostname(mc.kubeController, split.Spec.Service)
			apexMeshService := service.MeshService{
				Namespace:  svc.Namespace,
				Name:       svcName,
				Port:       svc.Port,
				TargetPort: svc.TargetPort,
				Protocol:   svc.Protocol,
			}

			if subdomain != "" {
				apexMeshService.Name = fmt.Sprintf("%s.%s", subdomain, svcName)
			}

			if newlyAdded := svcSet.Add(apexMeshService); newlyAdded {
				allServices = append(allServices, apexMeshService)
			}
		}
	}

	return allServices
}
