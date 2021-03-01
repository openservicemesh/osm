package catalog

import (
	"fmt"

	mapset "github.com/deckarep/golang-set"
	"github.com/pkg/errors"
	access "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/access/v1alpha3"

	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/kubernetes"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/trafficpolicy"
	"github.com/openservicemesh/osm/pkg/utils"
)

var wildCardRouteMatch trafficpolicy.HTTPRouteMatch = trafficpolicy.HTTPRouteMatch{
	PathRegex: constants.RegexMatchAll,
	Methods:   []string{constants.WildcardHTTPMethod},
}

// ListOutboundTrafficPolicies returns all outbound traffic policies
// 1. from service discovery for permissive mode
// 2. for the given service account from SMI Traffic Target and Traffic Split
func (mc *MeshCatalog) ListOutboundTrafficPolicies(downstreamIdentity service.K8sServiceAccount) []*trafficpolicy.OutboundTrafficPolicy {
	if mc.configurator.IsPermissiveTrafficPolicyMode() {
		outboundPolicies := []*trafficpolicy.OutboundTrafficPolicy{}
		mergedPolicies := trafficpolicy.MergeOutboundPolicies(outboundPolicies, mc.buildOutboundPermissiveModePolicies()...)
		outboundPolicies = mergedPolicies
		return outboundPolicies
	}

	outbound := mc.listOutboundPoliciesForTrafficTargets(downstreamIdentity)
	outboundPoliciesFromSplits := mc.listOutboundTrafficPoliciesForTrafficSplits(downstreamIdentity.Namespace)
	outbound = trafficpolicy.MergeOutboundPolicies(outbound, outboundPoliciesFromSplits...)

	return outbound
}

// ListInboundTrafficPolicies returns all inbound traffic policies
// 1. from service discovery for permissive mode
// 2. for the given service account and upstream services from SMI Traffic Target and Traffic Split
func (mc *MeshCatalog) ListInboundTrafficPolicies(upstreamIdentity service.K8sServiceAccount, upstreamServices []service.MeshService) []*trafficpolicy.InboundTrafficPolicy {
	if mc.configurator.IsPermissiveTrafficPolicyMode() {
		inboundPolicies := []*trafficpolicy.InboundTrafficPolicy{}
		for _, svc := range upstreamServices {
			inboundPolicies = trafficpolicy.MergeInboundPolicies(false, inboundPolicies, mc.buildInboundPermissiveModePolicies(svc)...)
		}
		return inboundPolicies
	}

	inbound := mc.listInboundPoliciesFromTrafficTargets(upstreamIdentity, upstreamServices)
	inboundPoliciesFRomSplits := mc.listInboundPoliciesForTrafficSplits(upstreamIdentity, upstreamServices)
	inbound = trafficpolicy.MergeInboundPolicies(false, inbound, inboundPoliciesFRomSplits...)
	return inbound
}

func (mc *MeshCatalog) listOutboundTrafficPoliciesForTrafficSplits(sourceNamespace string) []*trafficpolicy.OutboundTrafficPolicy {
	outboundPoliciesFromSplits := []*trafficpolicy.OutboundTrafficPolicy{}

	apexServices := mapset.NewSet()
	for _, split := range mc.meshSpec.ListTrafficSplits() {
		svc := service.MeshService{
			Name:      split.Spec.Service,
			Namespace: split.ObjectMeta.Namespace,
		}

		hostnames, err := mc.getServiceHostnames(svc, svc.Namespace == sourceNamespace)
		if err != nil {
			log.Error().Err(err).Msgf("Error getting service hostnames for apex service %v", svc)
			continue
		}
		policy := trafficpolicy.NewOutboundTrafficPolicy(buildPolicyName(svc, sourceNamespace == svc.Namespace), hostnames)

		rwc := trafficpolicy.RouteWeightedClusters{
			HTTPRouteMatch:   wildCardRouteMatch,
			WeightedClusters: mapset.NewSet(),
		}
		for _, backend := range split.Spec.Backends {
			ms := service.MeshService{Name: backend.Service, Namespace: split.ObjectMeta.Namespace}
			wc := service.WeightedCluster{
				ClusterName: service.ClusterName(ms.String()),
				Weight:      backend.Weight,
			}
			rwc.WeightedClusters.Add(wc)
		}
		policy.Routes = []*trafficpolicy.RouteWeightedClusters{&rwc}

		if apexServices.Contains(svc) {
			log.Error().Msgf("Skipping Traffic Split policy %s in namespaces %s as there is already a traffic split policy for apex service %v", split.Name, split.Namespace, svc)
		} else {
			outboundPoliciesFromSplits = append(outboundPoliciesFromSplits, policy)
			apexServices.Add(svc)
		}
	}
	return outboundPoliciesFromSplits
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
						servicePolicy.AddRule(*trafficpolicy.NewRouteWeightedCluster(routeMatch, weightedCluster), sourceServiceAccount)
					}
				}
				inboundPolicies = trafficpolicy.MergeInboundPolicies(false, inboundPolicies, servicePolicy)
			}
		}
	}
	return inboundPolicies
}

// ListAllowedOutboundServicesForIdentity list the services the given service account is allowed to initiate outbound connections to
func (mc *MeshCatalog) ListAllowedOutboundServicesForIdentity(identity service.K8sServiceAccount) []service.MeshService {
	if mc.configurator.IsPermissiveTrafficPolicyMode() {
		return mc.listMeshServices()
	}

	serviceSet := mapset.NewSet()
	for _, t := range mc.meshSpec.ListTrafficTargets() { // loop through all traffic targets
		for _, source := range t.Spec.Sources {
			if source.Name == identity.Name && source.Namespace == identity.Namespace { // found outbound
				destServices, err := mc.GetServicesForServiceAccount(service.K8sServiceAccount{
					Name:      t.Spec.Destination.Name,
					Namespace: t.Spec.Destination.Namespace,
				})
				if err != nil {
					log.Error().Err(err).Msgf("No Services found matching Service Account %s in Namespace %s", t.Spec.Destination.Name, t.Namespace)
					break
				}
				for _, destService := range destServices {
					serviceSet.Add(destService)
				}
				break
			}
		}
	}

	var allowedServices []service.MeshService
	for elem := range serviceSet.Iter() {
		allowedServices = append(allowedServices, elem.(service.MeshService))
	}
	return allowedServices
}

// getServiceHostnames returns a list of hostnames corresponding to the service.
// If the service is in the same namespace, it returns the shorthand hostname for the service that does not
// include its namespace, ex: bookstore, bookstore:80
func (mc *MeshCatalog) getServiceHostnames(meshService service.MeshService, sameNamespace bool) ([]string, error) {
	svc := mc.kubeController.GetService(meshService)
	if svc == nil {
		return nil, errors.Errorf("Error fetching service %q", meshService)
	}

	hostnames := kubernetes.GetHostnamesForService(svc, sameNamespace)
	return hostnames, nil
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
			serviceRoute := trafficpolicy.HTTPRouteMatch{}
			serviceRoute.PathRegex = trafficSpecsMatches.PathRegex
			serviceRoute.Methods = trafficSpecsMatches.Methods
			serviceRoute.Headers = trafficSpecsMatches.Headers
			if len(serviceRoute.Headers) != 0 {
				// When pathRegex and methods are not defined, the header filters are applied to any path and all HTTP methods
				if serviceRoute.PathRegex == "" {
					serviceRoute.PathRegex = constants.RegexMatchAll
				}
				if serviceRoute.Methods == nil {
					serviceRoute.Methods = []string{constants.WildcardHTTPMethod}
				}
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

func getDefaultWeightedClusterForService(meshService service.MeshService) service.WeightedCluster {
	return service.WeightedCluster{
		ClusterName: service.ClusterName(meshService.String()),
		Weight:      constants.ClusterWeightAcceptAll,
	}
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

// listMeshServices returns all services in the mesh
func (mc *MeshCatalog) listMeshServices() []service.MeshService {
	services := []service.MeshService{}
	for _, svc := range mc.kubeController.ListServices() {
		services = append(services, utils.K8sSvcToMeshSvc(svc))
	}
	return services
}

func (mc *MeshCatalog) getDestinationServicesFromTrafficTarget(t *access.TrafficTarget) ([]service.MeshService, error) {
	sa := service.K8sServiceAccount{
		Name:      t.Spec.Destination.Name,
		Namespace: t.Spec.Destination.Namespace,
	}
	destServices, err := mc.GetServicesForServiceAccount(sa)
	if err != nil {
		return nil, errors.Errorf("Error finding Services for Service Account %#v: %v", sa, err)
	}
	return destServices, nil
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
			servicePolicy.AddRule(*trafficpolicy.NewRouteWeightedCluster(routeMatch, weightedCluster), sourceServiceAccount)
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
	svcAccounts := mc.kubeController.ListServiceAccounts()

	// Build a rule for every service account in the mesh
	for _, svcAccount := range svcAccounts {
		sa := utils.SvcAccountToK8sSvcAccount(svcAccount)
		servicePolicy.AddRule(*trafficpolicy.NewRouteWeightedCluster(wildCardRouteMatch, weightedCluster), sa)
	}

	if len(servicePolicy.Rules) > 0 {
		inboundPolicies = append(inboundPolicies, servicePolicy)
	}
	return inboundPolicies
}

func (mc *MeshCatalog) buildOutboundPermissiveModePolicies() []*trafficpolicy.OutboundTrafficPolicy {
	outPolicies := []*trafficpolicy.OutboundTrafficPolicy{}

	k8sServices := mc.kubeController.ListServices()
	var destServices []service.MeshService
	for _, k8sService := range k8sServices {
		destServices = append(destServices, utils.K8sSvcToMeshSvc(k8sService))
	}

	for _, destService := range destServices {
		hostnames, err := mc.getServiceHostnames(destService, false)
		if err != nil {
			log.Error().Err(err).Msgf("Error getting service hostnames for service %s", destService)
			continue
		}

		weightedCluster := getDefaultWeightedClusterForService(destService)
		policy := trafficpolicy.NewOutboundTrafficPolicy(buildPolicyName(destService, false), hostnames)
		if err := policy.AddRoute(wildCardRouteMatch, weightedCluster); err != nil {
			log.Error().Err(err).Msgf("Error adding route to outbound policy in permissive mode for destination %s(%s)", destService.Name, destService.Namespace)
			continue
		}
		outPolicies = append(outPolicies, policy)
	}
	return outPolicies
}

func (mc *MeshCatalog) buildOutboundPolicies(source service.K8sServiceAccount, t *access.TrafficTarget) []*trafficpolicy.OutboundTrafficPolicy {
	outPolicies := []*trafficpolicy.OutboundTrafficPolicy{}

	// fetch services running workloads with destination service account
	destServices, err := mc.getDestinationServicesFromTrafficTarget(t)
	if err != nil {
		log.Error().Err(err).Msgf("Error resolving destination from traffic target %s (%s)", t.Name, t.Namespace)
		return outPolicies
	}

	// build an outbound traffic policy for each destination service
	for _, destService := range destServices {
		hostnames, err := mc.getServiceHostnames(destService, source.Namespace == destService.Namespace)
		if err != nil {
			log.Error().Err(err).Msgf("Error getting service hostnames for service %s", destService)
			continue
		}
		weightedCluster := getDefaultWeightedClusterForService(destService)

		policy := trafficpolicy.NewOutboundTrafficPolicy(buildPolicyName(destService, source.Namespace == destService.Namespace), hostnames)
		if err := policy.AddRoute(wildCardRouteMatch, weightedCluster); err != nil {
			log.Error().Err(err).Msgf("Error adding Route to outbound policy for source %s(%s) and destination %s (%s)", source.Name, source.Namespace, destService.Name, destService.Namespace)
			continue
		}

		outPolicies = append(outPolicies, policy)
	}
	return outPolicies
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
			inboundPolicies = trafficpolicy.MergeInboundPolicies(false, inboundPolicies, mc.buildInboundPolicies(t, svc)...)
		}
	}

	return inboundPolicies
}

// listOutboundPoliciesForTrafficTargets loops through all SMI Traffic Target resources and returns outbound traffic policies
// when the given service account matches a source in the Traffic Target resource
func (mc *MeshCatalog) listOutboundPoliciesForTrafficTargets(downstreamIdentity service.K8sServiceAccount) []*trafficpolicy.OutboundTrafficPolicy {
	outboundPolicies := []*trafficpolicy.OutboundTrafficPolicy{}

	for _, t := range mc.meshSpec.ListTrafficTargets() { // loop through all traffic targets
		if !isValidTrafficTarget(t) {
			continue
		}

		for _, source := range t.Spec.Sources {
			if source.Name == downstreamIdentity.Name && source.Namespace == downstreamIdentity.Namespace { // found outbound
				mergedPolicies := trafficpolicy.MergeOutboundPolicies(outboundPolicies, mc.buildOutboundPolicies(downstreamIdentity, t)...)
				outboundPolicies = mergedPolicies
				break
			}
		}
	}
	return outboundPolicies
}

// buildPolicyName creates a name for a policy associated with the given service
func buildPolicyName(svc service.MeshService, sameNamespace bool) string {
	name := svc.Name
	if !sameNamespace {
		return name + "." + svc.Namespace
	}
	return name
}
