package catalog

import (
	"fmt"
	"reflect"
	"strings"

	mapset "github.com/deckarep/golang-set"
	"github.com/pkg/errors"
	access "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/access/v1alpha3"
	corev1 "k8s.io/api/core/v1"

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

func (mc *MeshCatalog) listTrafficPoliciesForTrafficSplits(sourceNamespace string) []*trafficpolicy.OutboundTrafficPolicy {
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

// ListTrafficPoliciesForServiceAccount returns all inbound and outbound traffic policies related to the given service account
func (mc *MeshCatalog) ListTrafficPoliciesForServiceAccount(sa service.K8sServiceAccount) ([]*trafficpolicy.InboundTrafficPolicy, []*trafficpolicy.OutboundTrafficPolicy, error) {
	inbound, outbound, err := mc.listPoliciesFromTrafficTargets(sa)
	if err != nil {
		return nil, nil, err
	}

	outboundPoliciesFromSplits := mc.listTrafficPoliciesForTrafficSplits(sa.Namespace)
	outbound = trafficpolicy.MergeOutboundPolicies(outbound, outboundPoliciesFromSplits...)

	return inbound, outbound, nil
}

// ListTrafficPolicies returns all the traffic policies for a given service that Envoy proxy should be aware of.
func (mc *MeshCatalog) ListTrafficPolicies(service service.MeshService) ([]trafficpolicy.TrafficTarget, error) {
	log.Trace().Msgf("Listing traffic policies for service: %s", service)

	if mc.configurator.IsPermissiveTrafficPolicyMode() {
		// Build traffic policies from service discovery for allow-all policy
		trafficPolicies := mc.buildAllowAllTrafficPolicies(service)
		return trafficPolicies, nil
	}

	// Build traffic policies from SMI
	allRoutes, err := mc.getHTTPPathsPerRoute()
	if err != nil {
		log.Error().Err(err).Msgf("Error getting all paths per route while working on service %s", service)
		return nil, err
	}

	allTrafficPolicies, err := getTrafficPoliciesForService(mc, allRoutes, service)
	if err != nil {
		log.Error().Err(err).Msgf("Could not get all traffic policies")
		return nil, err
	}
	return allTrafficPolicies, nil
}

// This function returns the list of connected services.
// This is a bimodal function:
//   - it could list services that are allowed to connect to the given service (inbound)
//   - it could list services that the given service can connect to (outbound)
func (mc *MeshCatalog) getAllowedDirectionalServices(svc service.MeshService, directn trafficDirection) ([]service.MeshService, error) {
	allTrafficPolicies, err := mc.ListTrafficPolicies(svc)
	if err != nil {
		log.Error().Err(err).Msg("Failed listing traffic routes")
		return nil, err
	}

	allowedServicesSet := mapset.NewSet()

	for _, policy := range allTrafficPolicies {
		if directn == inbound {
			// we are looking for services that can connect to the given service
			if policy.Destination.Equals(svc) {
				allowedServicesSet.Add(policy.Source)
			}
		}

		if directn == outbound {
			// we are looking for services the given svc can connect to
			if policy.Source.Equals(svc) {
				allowedServicesSet.Add(policy.Destination)
			}
		}
	}

	// Convert the set of interfaces to a list of namespaced services
	var allowedServices []service.MeshService
	for svc := range allowedServicesSet.Iter() {
		allowedServices = append(allowedServices, svc.(service.MeshService))
	}

	msg := map[trafficDirection]string{
		inbound:  "Allowed inbound services for destination service %q: %+v",
		outbound: "Allowed outbound services from source %q: %+v",
	}[directn]

	log.Debug().Msgf(msg, svc, allowedServices)

	return allowedServices, nil
}

// ListAllowedInboundServices lists the inbound services allowed to connect to the given service.
func (mc *MeshCatalog) ListAllowedInboundServices(destinationService service.MeshService) ([]service.MeshService, error) {
	return mc.getAllowedDirectionalServices(destinationService, inbound)
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

//GetWeightedClusterForService returns the weighted cluster for a given service
func (mc *MeshCatalog) GetWeightedClusterForService(svc service.MeshService) (service.WeightedCluster, error) {
	log.Trace().Msgf("Looking for weighted cluster for service %s", svc)

	if mc.configurator.IsPermissiveTrafficPolicyMode() {
		return getDefaultWeightedClusterForService(svc), nil
	}

	// Retrieve the weighted clusters from traffic split
	servicesList := mc.meshSpec.ListTrafficSplitServices()
	for _, activeService := range servicesList {
		if activeService.Service == svc {
			return service.WeightedCluster{
				ClusterName: service.ClusterName(activeService.Service.String()),
				Weight:      activeService.Weight,
			}, nil
		}
	}

	// Use a default weighted cluster as an SMI TrafficSplit policy is not defined for the service
	return getDefaultWeightedClusterForService(svc), nil
}

// GetResolvableHostnamesForUpstreamService returns the hostnames over which an upstream service is accessible from a downstream service
// The hostname is the FQDN for the service, and can include ports as well.
// Ex. bookstore.default, bookstore.default:80, bookstore.default.svc, bookstore.default.svc:80 etc.
// TODO : remove as a part of routes refactor (#2397)
func (mc *MeshCatalog) GetResolvableHostnamesForUpstreamService(downstream, upstream service.MeshService) ([]string, error) {
	sameNamespace := downstream.Namespace == upstream.Namespace
	var svcHostnames []string

	if mc.configurator.IsPermissiveTrafficPolicyMode() {
		hostnames, err := mc.getServiceHostnames(upstream, sameNamespace)
		if err != nil {
			log.Error().Err(err).Msgf("Error getting service hostnames for upstream service %s", upstream)
			return svcHostnames, err
		}
		return hostnames, nil
	}

	// If this service is referenced in a traffic split
	// Retrieve the domain name from traffic split root service
	servicesList := mc.meshSpec.ListTrafficSplitServices()
	for _, activeService := range servicesList {
		if activeService.Service == upstream {
			log.Trace().Msgf("Getting hostnames for upstream service %s", upstream)
			rootServiceName := kubernetes.GetServiceFromHostname(activeService.RootService)
			rootMeshService := service.MeshService{
				Namespace: upstream.Namespace,
				Name:      rootServiceName,
			}
			hostnames, err := mc.getServiceHostnames(rootMeshService, sameNamespace)
			if err != nil {
				log.Error().Err(err).Msgf("Error getting service hostnames for Apex service %s", rootMeshService)
				return svcHostnames, err
			}
			svcHostnames = append(svcHostnames, hostnames...)
		}
	}

	// The hostnames for this service are the Kubernetes service DNS names.
	hostnames, err := mc.getServiceHostnames(upstream, sameNamespace)
	if err != nil {
		log.Error().Err(err).Msgf("Error getting service hostnames for upstream service %s", upstream)
		return svcHostnames, err
	}

	svcHostnames = append(svcHostnames, hostnames...)
	return svcHostnames, nil
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

// hashSrcDstService returns a hash for the source and destination MeshService
func hashSrcDstService(src service.MeshService, dst service.MeshService) string {
	return fmt.Sprintf("%s:%s", src, dst)
}

// getTrafficTargetFromSrcDstHash returns a TrafficTarget object given a hash computed by 'hashSrcDstService', its name and routes
func getTrafficTargetFromSrcDstHash(hash string, name string, httpRoutes []trafficpolicy.HTTPRouteMatch) trafficpolicy.TrafficTarget {
	s := strings.Split(hash, ":")
	src, _ := service.UnmarshalMeshService(s[0])
	dst, _ := service.UnmarshalMeshService(s[1])

	return trafficpolicy.TrafficTarget{
		Name:             name,
		Source:           *src,
		Destination:      *dst,
		HTTPRouteMatches: httpRoutes,
	}
}

// getTrafficPoliciesForService returns a list of TrafficTarget policies associated with a given MeshService.
// The function consolidates all the routes between a source and destination in a single TrafficTarget object.
func getTrafficPoliciesForService(mc *MeshCatalog, routePolicies map[trafficpolicy.TrafficSpecName]map[trafficpolicy.TrafficSpecMatchName]trafficpolicy.HTTPRouteMatch, meshService service.MeshService) ([]trafficpolicy.TrafficTarget, error) {
	// 'srcDstTrafficTargetMap' is used to consolidate all routes from a source to a destination service.
	// For the same source to destination if multiple routes are specified, all the routes are
	// a part of a single TrafficTarget associated with that source and destination.
	srcDstTrafficTargetMap := make(map[string]trafficpolicy.TrafficTarget)

	// 'matchedTrafficTargets' is the list of all computed TrafficTarget policies that the given 'meshService`
	// is a part of.
	var matchedTrafficTargets []trafficpolicy.TrafficTarget

	for _, trafficTargets := range mc.meshSpec.ListTrafficTargets() {
		log.Trace().Msgf("Discovered TrafficTarget resource: %s/%s", trafficTargets.Namespace, trafficTargets.Name)
		if !isValidTrafficTarget(trafficTargets) {
			log.Error().Msgf("TrafficTarget %s/%s is not valid; Skipping...", trafficTargets.Namespace, trafficTargets.Name)
			continue
		}

		for _, trafficSources := range trafficTargets.Spec.Sources {
			trafficTargetPermutations, err := mc.listTrafficTargetPermutations(*trafficTargets, trafficSources, trafficTargets.Spec.Destination)
			if err != nil {
				log.Error().Msgf("Could not list services for TrafficTarget %s/%s", trafficTargets.Namespace, trafficTargets.Name)
				return nil, err
			}
			for _, trafficTarget := range trafficTargetPermutations {
				var httpRoutes []trafficpolicy.HTTPRouteMatch // Keeps track of all the routes from a source to a destination service

				for _, trafficTargetSpecs := range trafficTargets.Spec.Rules {
					if trafficTargetSpecs.Kind != httpRouteGroupKind {
						// If kind does not match HTTPRouteGroup, skip it as it is not required to build HTTP traffic policies
						continue
					}

					specKey := mc.getTrafficSpecName(trafficTargetSpecs.Kind, trafficTargets.Namespace, trafficTargetSpecs.Name)
					routePoliciesMatched, matchFound := routePolicies[specKey]
					if !matchFound {
						log.Error().Msgf("TrafficTarget %s/%s could not find a TrafficSpec %s", trafficTargets.Namespace, trafficTargets.Name, specKey)
						return nil, errNoTrafficSpecFoundForTrafficPolicy
					}
					if len(trafficTargetSpecs.Matches) == 0 {
						// This TrafficTarget does not match against a specific route match criteria defined in the
						// associated traffic spec resource, so consider all the routes to match against.
						for _, routePolicy := range routePoliciesMatched {
							// Consider this route for the current traffic target object being evaluated
							httpRoutes = append(httpRoutes, routePolicy)
						}
					} else {
						// This TrafficTarget has a match criteria specified to match against specific routes, so
						// only consider those routes that match.
						for _, specMatchesName := range trafficTargetSpecs.Matches {
							routePolicy, matchFound := routePoliciesMatched[trafficpolicy.TrafficSpecMatchName(specMatchesName)]
							if !matchFound {
								log.Error().Msgf("TrafficTarget %s/%s could not find a TrafficSpec %s with match name %s", trafficTargets.Namespace, trafficTargets.Name, specKey, specMatchesName)
								return nil, errNoTrafficSpecFoundForTrafficPolicy
							}
							// Consider this route for the current traffic target object being evaluated
							httpRoutes = append(httpRoutes, routePolicy)
						}
					}
				}

				if trafficTarget.Source.Equals(meshService) || trafficTarget.Destination.Equals(meshService) {
					// The given meshService is a source or destination for this trafficTarget, so add
					// it to the list of traffic targets associated with this service.
					srcDstServiceHash := hashSrcDstService(trafficTarget.Source, trafficTarget.Destination)
					srcDstTrafficTarget := getTrafficTargetFromSrcDstHash(srcDstServiceHash, trafficTarget.Name, httpRoutes)
					srcDstTrafficTargetMap[srcDstServiceHash] = srcDstTrafficTarget
				}
			}
		}
	}

	for _, trafficTarget := range srcDstTrafficTargetMap {
		matchedTrafficTargets = append(matchedTrafficTargets, trafficTarget)
	}

	log.Debug().Msgf("Traffic policies for service %s: %+v", meshService, matchedTrafficTargets)
	return matchedTrafficTargets, nil
}

func (mc *MeshCatalog) buildAllowAllTrafficPolicies(service service.MeshService) []trafficpolicy.TrafficTarget {
	services := mc.kubeController.ListServices()

	var trafficTargets []trafficpolicy.TrafficTarget
	for _, source := range services {
		for _, destination := range services {
			if reflect.DeepEqual(source, destination) {
				continue
			}
			allowTrafficTarget := mc.buildAllowPolicyForSourceToDest(source, destination)
			trafficTargets = append(trafficTargets, allowTrafficTarget)
		}
	}
	log.Debug().Msgf("All traffic policies for service %s : %v", service.String(), trafficTargets)
	return trafficTargets
}

func (mc *MeshCatalog) buildAllowPolicyForSourceToDest(source *corev1.Service, destination *corev1.Service) trafficpolicy.TrafficTarget {
	srcMeshSvc := utils.K8sSvcToMeshSvc(source)
	dstMeshSvc := utils.K8sSvcToMeshSvc(destination)
	return trafficpolicy.TrafficTarget{
		Name:             utils.GetTrafficTargetName("", srcMeshSvc, dstMeshSvc),
		Destination:      dstMeshSvc,
		Source:           srcMeshSvc,
		HTTPRouteMatches: []trafficpolicy.HTTPRouteMatch{wildCardRouteMatch},
	}
}

func getDefaultWeightedClusterForService(meshService service.MeshService) service.WeightedCluster {
	return service.WeightedCluster{
		ClusterName: service.ClusterName(meshService.String()),
		Weight:      constants.ClusterWeightAcceptAll,
	}
}

// listTrafficTargetPermutations creates a list of TrafficTargets for each source and destination pair.
func (mc *MeshCatalog) listTrafficTargetPermutations(trafficTarget access.TrafficTarget, src access.IdentityBindingSubject, dest access.IdentityBindingSubject) ([]trafficpolicy.TrafficTarget, error) {
	sourceServiceAccount := service.K8sServiceAccount{
		Namespace: src.Namespace,
		Name:      src.Name,
	}

	srcServiceList, srcErr := mc.GetServicesForServiceAccount(sourceServiceAccount)
	if srcErr != nil {
		log.Error().Msgf("TrafficTarget %s/%s could not get source services for service account %s", trafficTarget.Namespace, trafficTarget.Name, sourceServiceAccount.String())
		return nil, srcErr
	}

	dstNamespacedServiceAcc := service.K8sServiceAccount{
		Namespace: dest.Namespace,
		Name:      dest.Name,
	}
	destServiceList, destErr := mc.GetServicesForServiceAccount(dstNamespacedServiceAcc)
	if destErr != nil {
		log.Error().Msgf("TrafficTarget %s/%s could not get destination services for service account %s", trafficTarget.Namespace, trafficTarget.Name, dstNamespacedServiceAcc.String())
		return nil, destErr
	}

	trafficPolicies := make([]trafficpolicy.TrafficTarget, 0, len(srcServiceList)*len(destServiceList))

	for _, destService := range destServiceList {
		for _, srcService := range srcServiceList {
			trafficTarget := trafficpolicy.TrafficTarget{
				Name:        utils.GetTrafficTargetName(trafficTarget.Name, srcService, destService),
				Destination: destService,
				Source:      srcService,
			}
			trafficPolicies = append(trafficPolicies, trafficTarget)
		}
	}

	return trafficPolicies, nil
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

func (mc *MeshCatalog) buildInboundPolicies(t *access.TrafficTarget) []*trafficpolicy.InboundTrafficPolicy {
	inboundPolicies := []*trafficpolicy.InboundTrafficPolicy{}

	// fetch services running workloads with destination service account
	destServices, err := mc.getDestinationServicesFromTrafficTarget(t)
	if err != nil {
		log.Error().Err(err).Msgf("Error resolving destination from traffic target %s (%s)", t.Name, t.Namespace)
		return inboundPolicies
	}

	// fetch all routes referenced in traffic target
	routeMatches, err := mc.routesFromRules(t.Spec.Rules, t.Namespace)
	if err != nil {
		log.Error().Err(err).Msgf("Error finding route matches from TrafficTarget %s in namespace %s", t.Name, t.Namespace)
		return inboundPolicies
	}

	for _, destService := range destServices {
		hostnames, err := mc.getServiceHostnames(destService, true)
		if err != nil {
			continue
		}

		servicePolicy := trafficpolicy.NewInboundTrafficPolicy(buildPolicyName(destService, false), hostnames)

		weightedCluster := getDefaultWeightedClusterForService(destService)

		for _, sourceServiceAccount := range trafficTargetIdentitiesToSvcAccounts(t.Spec.Sources) {
			for _, routeMatch := range routeMatches {
				servicePolicy.AddRule(*trafficpolicy.NewRouteWeightedCluster(routeMatch, weightedCluster), sourceServiceAccount)
			}
		}

		if len(servicePolicy.Rules) > 0 {
			inboundPolicies = append(inboundPolicies, servicePolicy)
		}
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

func (mc *MeshCatalog) buildOutboundPermissiveModePolicies(srcServices []service.MeshService) []*trafficpolicy.OutboundTrafficPolicy {
	outPolicies := []*trafficpolicy.OutboundTrafficPolicy{}

	k8sServices := mc.kubeController.ListServices()
	var destServices []service.MeshService
	for _, k8sService := range k8sServices {
		destServices = append(destServices, utils.K8sSvcToMeshSvc(k8sService))
	}

	// For permissive mode build an outbound policy to every service in the mesh except to itself
	destServices = difference(destServices, srcServices)

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

// listPoliciesFromTrafficTargets loops through all SMI Traffic Target resources and returns inbound and outbound traffic policies
// based on when the given service account matches a destination or source in the Traffic Target resource
func (mc *MeshCatalog) listPoliciesFromTrafficTargets(sa service.K8sServiceAccount) ([]*trafficpolicy.InboundTrafficPolicy, []*trafficpolicy.OutboundTrafficPolicy, error) {
	inboundPolicies := []*trafficpolicy.InboundTrafficPolicy{}
	outboundPolicies := []*trafficpolicy.OutboundTrafficPolicy{}

	for _, t := range mc.meshSpec.ListTrafficTargets() { // loop through all traffic targets
		if !isValidTrafficTarget(t) {
			continue
		}

		if t.Spec.Destination.Name == sa.Name { // found inbound
			inboundPolicies = trafficpolicy.MergeInboundPolicies(false, inboundPolicies, mc.buildInboundPolicies(t)...)
		}

		for _, source := range t.Spec.Sources {
			if source.Name == sa.Name && source.Namespace == sa.Namespace { // found outbound
				mergedPolicies := trafficpolicy.MergeOutboundPolicies(outboundPolicies, mc.buildOutboundPolicies(sa, t)...)
				outboundPolicies = mergedPolicies
				break
			}
		}
	}
	return inboundPolicies, outboundPolicies, nil
}

// ListPoliciesForPermissiveMode builds inbound and outbound traffic policies from service discovery
func (mc *MeshCatalog) ListPoliciesForPermissiveMode(services []service.MeshService) ([]*trafficpolicy.InboundTrafficPolicy, []*trafficpolicy.OutboundTrafficPolicy, error) {
	inboundPolicies := []*trafficpolicy.InboundTrafficPolicy{}
	outboundPolicies := []*trafficpolicy.OutboundTrafficPolicy{}

	for _, svc := range services {
		inboundPolicies = trafficpolicy.MergeInboundPolicies(false, inboundPolicies, mc.buildInboundPermissiveModePolicies(svc)...)
	}

	mergedPolicies := trafficpolicy.MergeOutboundPolicies(outboundPolicies, mc.buildOutboundPermissiveModePolicies(services)...)
	outboundPolicies = mergedPolicies
	return inboundPolicies, outboundPolicies, nil
}

// buildPolicyName creates a name for a policy associated with the given service
func buildPolicyName(svc service.MeshService, sameNamespace bool) string {
	name := svc.Name
	if !sameNamespace {
		return name + "." + svc.Namespace
	}
	return name
}

// difference returns the elements from largerSlice that are not found in smallerSlice
func difference(largerSlice, smallerSlice []service.MeshService) []service.MeshService {
	mb := make(map[service.MeshService]bool, len(largerSlice))
	for _, x := range largerSlice {
		mb[x] = false
	}
	var diff []service.MeshService
	for _, x := range smallerSlice {
		if _, found := mb[x]; found {
			mb[x] = true
		}
	}
	for key, val := range mb {
		if !val {
			diff = append(diff, key)
		}
	}
	return diff
}
