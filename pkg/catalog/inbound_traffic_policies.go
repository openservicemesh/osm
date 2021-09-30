package catalog

import (
	"fmt"

	mapset "github.com/deckarep/golang-set"
	access "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/access/v1alpha3"

	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/errcode"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/k8s"
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

	// Build configurations per upstream service
	for _, upstreamSvc := range upstreamServices {
		// ---
		// Create local cluster configs for this upstram service
		clusterConfigForSvc := &trafficpolicy.MeshClusterConfig{
			Name:    upstreamSvc.EnvoyLocalClusterName(),
			Service: upstreamSvc,
			Address: constants.LocalhostIPAddress,
			Port:    uint32(upstreamSvc.TargetPort),
		}
		clusterConfigs = append(clusterConfigs, clusterConfigForSvc)

		// ---
		// Create a TrafficMatch for this upstream servic.
		// The TrafficMatch will be used by LDS to program a filter chain match
		// for this upstream service on the upstream server to accept inbound
		// traffic.
		trafficMatchForUpstreamSvc := &trafficpolicy.TrafficMatch{
			Name:                fmt.Sprintf("%s_%d_%s", upstreamSvc, upstreamSvc.TargetPort, upstreamSvc.Protocol),
			DestinationPort:     int(upstreamSvc.TargetPort),
			DestinationProtocol: upstreamSvc.Protocol,
		}
		trafficMatches = append(trafficMatches, trafficMatchForUpstreamSvc)

		// Build the HTTP route configs for this service and port combination.
		// If the port's protocol corresponds to TCP, we can skip this step
		if upstreamSvc.Protocol == constants.ProtocolTCP {
			continue
		}
		// ---
		// Build the HTTP route configs per port
		// Each upstream service accepts traffic from downstreams on a list of allowed routes.
		// The routes are derived from SMI TrafficTarget and TrafficSplit policies in SMI mode,
		// and are wildcarded in permissive mode. The downstreams that can access this upstream
		// on the configured routes is also determined based on the traffic policy mode.
		inboundTrafficPolicies := mc.getInboundTrafficPoliciesForUpstream(upstreamIdentity, upstreamSvc, permissiveMode, trafficTargets)
		routeConfigPerPort[int(upstreamSvc.TargetPort)] = append(routeConfigPerPort[int(upstreamSvc.TargetPort)], inboundTrafficPolicies...)
	}

	return &trafficpolicy.InboundMeshTrafficPolicy{
		TrafficMatches:          trafficMatches,
		ClustersConfigs:         clusterConfigs,
		HTTPRouteConfigsPerPort: routeConfigPerPort,
	}
}

func (mc *MeshCatalog) getInboundTrafficPoliciesForUpstream(upstreamIdentity identity.ServiceIdentity, upstreamSvc service.MeshService, permissiveMode bool, trafficTargets []*access.TrafficTarget) []*trafficpolicy.InboundTrafficPolicy {
	var inboundTrafficPolicies []*trafficpolicy.InboundTrafficPolicy
	var inboundPolicyForUpstreamSvc *trafficpolicy.InboundTrafficPolicy

	if permissiveMode {
		// Add a wildcard HTTP route that allows any downstream client to access the upstream service
		hostnames := k8s.GetHostnamesForService(upstreamSvc, true /* local namespace FQDN should always be allowed for inbound routes*/)
		inboundPolicyForUpstreamSvc = trafficpolicy.NewInboundTrafficPolicy(upstreamSvc.FQDN(), hostnames)
		localCluster := service.WeightedCluster{
			ClusterName: service.ClusterName(upstreamSvc.EnvoyLocalClusterName()),
			Weight:      constants.ClusterWeightAcceptAll,
		}
		inboundPolicyForUpstreamSvc.AddRule(*trafficpolicy.NewRouteWeightedCluster(trafficpolicy.WildCardRouteMatch, []service.WeightedCluster{localCluster}), identity.WildcardServiceIdentity)
	} else {
		// Build the HTTP routes from SMI TrafficTarget and HTTPRouteGroup configurations
		inboundPolicyForUpstreamSvc = mc.buildInboundHTTPPolicyFromTrafficTarget(upstreamIdentity, upstreamSvc, trafficTargets)
	}
	inboundTrafficPolicies = append(inboundTrafficPolicies, inboundPolicyForUpstreamSvc)

	// If this upstream service is a backend for an apex service specified in a TrafficSplit configuration,
	// downstream clients are allowed to access this upstream using the hostnames of the corresponding upstream
	// apex service. To allow this, add additional routes corresponding to the apex services for this upstream backend.
	// The routes corresponding to the apex service hostnames must enforce the same rules as the the
	// route built for this `upstreamSvc`.
	inboundTrafficPolicies = append(inboundTrafficPolicies, mc.getInboundTrafficPoliciesFromSplit(upstreamSvc, inboundPolicyForUpstreamSvc.Rules)...)

	return inboundTrafficPolicies
}

func (mc *MeshCatalog) buildInboundHTTPPolicyFromTrafficTarget(upstreamIdentity identity.ServiceIdentity, upstreamSvc service.MeshService, trafficTargets []*access.TrafficTarget) *trafficpolicy.InboundTrafficPolicy {
	hostnames := k8s.GetHostnamesForService(upstreamSvc, true /* local namespace FQDN should always be allowed for inbound routes*/)
	inboundPolicy := trafficpolicy.NewInboundTrafficPolicy(upstreamSvc.FQDN(), hostnames)
	localCluster := service.WeightedCluster{
		ClusterName: service.ClusterName(upstreamSvc.EnvoyLocalClusterName()),
		Weight:      constants.ClusterWeightAcceptAll,
	}

	var routingRules []*trafficpolicy.Rule
	// From each TrafficTarget and HTTPRouteGroup configuration associated with this service, build routes for it.
	for _, trafficTarget := range trafficTargets {
		rules := mc.getRoutingRulesFromTrafficTarget(*trafficTarget, upstreamSvc, localCluster)
		// Multiple TrafficTarget objects can reference the same route, in which case such routes
		// need to be merged to create a single route that includes all the downstream client identities
		// this route is authorized for.
		routingRules = trafficpolicy.MergeRules(routingRules, rules)
	}
	inboundPolicy.Rules = routingRules

	return inboundPolicy
}

func (mc *MeshCatalog) getRoutingRulesFromTrafficTarget(trafficTarget access.TrafficTarget, upstreamSvc service.MeshService, routingCluster service.WeightedCluster) []*trafficpolicy.Rule {
	// Compute the HTTP route matches associated with the given TrafficTarget object
	httpRouteMatches, err := mc.routesFromRules(trafficTarget.Spec.Rules, trafficTarget.Namespace)
	if err != nil {
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrFetchingSMIHTTPRouteGroupForTrafficTarget)).
			Msgf("Error finding route matches from TrafficTarget %s in namespace %s", trafficTarget.Name, trafficTarget.Namespace)
		return nil
	}

	// Compute the allowed downstream service identities for the given TrafficTarget object
	allowedDownstreamIdentities := mapset.NewSet()
	for _, source := range trafficTarget.Spec.Sources {
		sourceSvcIdentity := trafficTargetIdentityToSvcAccount(source).ToServiceIdentity()
		allowedDownstreamIdentities.Add(sourceSvcIdentity)
	}

	var routingRules []*trafficpolicy.Rule
	for _, httpRouteMatch := range httpRouteMatches {
		rule := &trafficpolicy.Rule{
			Route:                    *trafficpolicy.NewRouteWeightedCluster(httpRouteMatch, []service.WeightedCluster{routingCluster}),
			AllowedServiceIdentities: allowedDownstreamIdentities,
		}
		routingRules = append(routingRules, rule)
	}

	return routingRules
}

func (mc *MeshCatalog) getInboundTrafficPoliciesFromSplit(upstreamSvc service.MeshService, routingRules []*trafficpolicy.Rule) []*trafficpolicy.InboundTrafficPolicy {
	// Retrieve all the traffic splits for which this service (upstreamSvc) is a backend.
	// HTTP routes to be able to access the apex services corresponding to this backend
	// will be built, matching the same routing rules enforced on the backend.
	var inboundPolicies []*trafficpolicy.InboundTrafficPolicy
	apexServiceSet := mapset.NewSet()
	trafficSplits := mc.meshSpec.ListTrafficSplits(smi.WithTrafficSplitBackendService(upstreamSvc))

	for _, split := range trafficSplits {
		apexMeshService := service.MeshService{
			Namespace:  upstreamSvc.Namespace,
			Name:       k8s.GetServiceFromHostname(split.Spec.Service),
			Port:       upstreamSvc.Port,
			TargetPort: upstreamSvc.TargetPort,
			Protocol:   upstreamSvc.Protocol,
		}
		// If apex service is same as the upstream service, ignore it because
		// a route for the service already exists. This can happen if the upstream
		// service is listed as a backend for itself in a traffic split policy.
		if apexMeshService == upstreamSvc {
			continue
		}
		apexServiceSet.Add(apexMeshService)
	}

	for svc := range apexServiceSet.Iter() {
		apexSvc := svc.(service.MeshService)
		hostnames := k8s.GetHostnamesForService(apexSvc, true /* local namespace FQDN should always be allowed for inbound routes*/)
		inboundPolicyForApexSvc := trafficpolicy.NewInboundTrafficPolicy(apexSvc.FQDN(), hostnames)
		inboundPolicyForApexSvc.Rules = routingRules
		inboundPolicies = append(inboundPolicies, inboundPolicyForApexSvc)
	}

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
		trafficSpecName := mc.getTrafficSpecName(smi.HTTPRouteGroupKind, trafficTargetNamespace, rule.Name)
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
		specKey := mc.getTrafficSpecName(smi.HTTPRouteGroupKind, trafficSpecs.Namespace, trafficSpecs.Name)
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
