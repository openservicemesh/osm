package catalog

import (
	"fmt"
	"reflect"
	"strings"

	mapset "github.com/deckarep/golang-set"
	"github.com/pkg/errors"
	target "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/access/v1alpha2"
	corev1 "k8s.io/api/core/v1"

	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/kubernetes"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/trafficpolicy"
	"github.com/openservicemesh/osm/pkg/utils"
)

const (
	//HTTPTraffic specifies HTTP Traffic Policy
	HTTPTraffic = "HTTPRouteGroup"
)

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

	log.Trace().Msgf(msg, svc, allowedServices)

	return allowedServices, nil
}

// ListAllowedInboundServices lists the inbound services allowed to connect to the given service.
func (mc *MeshCatalog) ListAllowedInboundServices(destinationService service.MeshService) ([]service.MeshService, error) {
	return mc.getAllowedDirectionalServices(destinationService, inbound)
}

// ListAllowedOutboundServices lists the services the given service is allowed outbound connections to.
func (mc *MeshCatalog) ListAllowedOutboundServices(sourceService service.MeshService) ([]service.MeshService, error) {
	return mc.getAllowedDirectionalServices(sourceService, outbound)
}

//GetWeightedClusterForService returns the weighted cluster for a given service
func (mc *MeshCatalog) GetWeightedClusterForService(svc service.MeshService) (service.WeightedCluster, error) {
	log.Trace().Msgf("Finding weighted cluster for service %s", svc)

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

// hostnamesTostr returns a comma separated string of hostnames from the list
func hostnamesTostr(hostnames []string) string {
	return strings.Join(hostnames, ",")
}

// GetResolvableHostnamesForUpstreamService returns the hostnames over which an upstream service is accessible from a downstream service
// The hostname is the FQDN for the service, and can include ports as well.
// Ex. bookstore.default, bookstore.default:80, bookstore.default.svc, bookstore.default.svc:80 etc.
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
			log.Trace().Msgf("Getting hostnames for service %s", upstream)
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

func (mc *MeshCatalog) getHTTPPathsPerRoute() (map[trafficpolicy.TrafficSpecName]map[trafficpolicy.TrafficSpecMatchName]trafficpolicy.HTTPRoute, error) {
	routePolicies := make(map[trafficpolicy.TrafficSpecName]map[trafficpolicy.TrafficSpecMatchName]trafficpolicy.HTTPRoute)
	for _, trafficSpecs := range mc.meshSpec.ListHTTPTrafficSpecs() {
		log.Debug().Msgf("Discovered TrafficSpec resource: %s/%s", trafficSpecs.Namespace, trafficSpecs.Name)
		if trafficSpecs.Spec.Matches == nil {
			log.Error().Msgf("TrafficSpec %s/%s has no matches in route; Skipping...", trafficSpecs.Namespace, trafficSpecs.Name)
			continue
		}

		// since this method gets only specs related to HTTPRouteGroups added HTTPTraffic to the specKey by default
		specKey := mc.getTrafficSpecName(HTTPTraffic, trafficSpecs.Namespace, trafficSpecs.Name)
		routePolicies[specKey] = make(map[trafficpolicy.TrafficSpecMatchName]trafficpolicy.HTTPRoute)
		for _, trafficSpecsMatches := range trafficSpecs.Spec.Matches {
			serviceRoute := trafficpolicy.HTTPRoute{}
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
func getTrafficTargetFromSrcDstHash(hash string, name string, httpRoutes []trafficpolicy.HTTPRoute) trafficpolicy.TrafficTarget {
	s := strings.Split(hash, ":")
	src, _ := service.UnmarshalMeshService(s[0])
	dst, _ := service.UnmarshalMeshService(s[1])

	return trafficpolicy.TrafficTarget{
		Name:        name,
		Source:      *src,
		Destination: *dst,
		HTTPRoutes:  httpRoutes,
	}
}

// getTrafficPoliciesForService returns a list of TrafficTarget policies associated with a given MeshService.
// The function consolidates all the routes between a source and destination in a single TrafficTarget object.
func getTrafficPoliciesForService(mc *MeshCatalog, routePolicies map[trafficpolicy.TrafficSpecName]map[trafficpolicy.TrafficSpecMatchName]trafficpolicy.HTTPRoute, meshService service.MeshService) ([]trafficpolicy.TrafficTarget, error) {
	// 'srcDstTrafficTargetMap' is used to consolidate all routes from a source to a destination service.
	// For the same source to destination if multiple routes are specified, all the routes are
	// a part of a single TrafficTarget associated with that source and destination.
	srcDstTrafficTargetMap := make(map[string]trafficpolicy.TrafficTarget)

	// 'matchedTrafficTargets' is the list of all computed TrafficTarget policies that the given 'meshService`
	// is a part of.
	var matchedTrafficTargets []trafficpolicy.TrafficTarget

	for _, trafficTargets := range mc.meshSpec.ListTrafficTargets() {
		log.Debug().Msgf("Discovered TrafficTarget resource: %s/%s", trafficTargets.Namespace, trafficTargets.Name)
		if !isValidTrafficTarget(trafficTargets) {
			log.Error().Msgf("TrafficTarget %s/%s has no spec routes; Skipping...", trafficTargets.Namespace, trafficTargets.Name)
			continue
		}

		for _, trafficSources := range trafficTargets.Spec.Sources {
			trafficTargetPermutations, err := mc.listTrafficTargetPermutations(*trafficTargets, trafficSources, trafficTargets.Spec.Destination)
			if err != nil {
				log.Error().Msgf("Could not list services for TrafficTarget %s/%s", trafficTargets.Namespace, trafficTargets.Name)
				return nil, err
			}
			for _, trafficTarget := range trafficTargetPermutations {
				var httpRoutes []trafficpolicy.HTTPRoute // Keeps track of all the routes from a source to a destination service

				for _, trafficTargetSpecs := range trafficTargets.Spec.Rules {
					if trafficTargetSpecs.Kind != HTTPTraffic {
						log.Error().Msgf("TrafficTarget %s/%s has Spec Kind %s which isn't supported for now; Skipping...", trafficTargets.Namespace, trafficTargets.Name, trafficTargetSpecs.Kind)
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
	log.Trace().Msgf("all traffic policies: %v", trafficTargets)
	return trafficTargets
}

func (mc *MeshCatalog) buildAllowPolicyForSourceToDest(source *corev1.Service, destination *corev1.Service) trafficpolicy.TrafficTarget {
	allowAllRoute := trafficpolicy.HTTPRoute{
		PathRegex: constants.RegexMatchAll,
		Methods:   []string{constants.WildcardHTTPMethod},
	}
	srcMeshSvc := utils.K8sSvcToMeshSvc(source)
	dstMeshSvc := utils.K8sSvcToMeshSvc(destination)
	return trafficpolicy.TrafficTarget{
		Name:        utils.GetTrafficTargetName("", srcMeshSvc, dstMeshSvc),
		Destination: dstMeshSvc,
		Source:      srcMeshSvc,
		HTTPRoutes:  []trafficpolicy.HTTPRoute{allowAllRoute},
	}
}

func getDefaultWeightedClusterForService(meshService service.MeshService) service.WeightedCluster {
	return service.WeightedCluster{
		ClusterName: service.ClusterName(meshService.String()),
		Weight:      constants.ClusterWeightAcceptAll,
	}
}

// listTrafficTargetPermutations creates a list of TrafficTargets for each source and destination pair.
func (mc *MeshCatalog) listTrafficTargetPermutations(trafficTarget target.TrafficTarget, src target.IdentityBindingSubject, dest target.IdentityBindingSubject) ([]trafficpolicy.TrafficTarget, error) {
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
//	http routes (trafficpolicy.HTTPRoute)
func (mc *MeshCatalog) routesFromRules(rules []target.TrafficTargetRule, trafficTargetNamespace string) ([]trafficpolicy.HTTPRoute, error) {
	routes := []trafficpolicy.HTTPRoute{}

	specMatchRoute, err := mc.getHTTPPathsPerRoute() // returns map[traffic_spec_name]map[match_name]trafficpolicy.HTTPRoute
	if err != nil {
		return nil, err
	}

	if len(specMatchRoute) == 0 {
		log.Debug().Msg("No elements in map[traffic_spec_name]map[match name]trafficpolicyHTTPRoute")
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

// GetServicesForServiceAccounts returns a list of services corresponding to a list service accounts
//	TODO: Consider merging this function and mc.GetServicesForServiceAccount in future (#2038)
func (mc *MeshCatalog) GetServicesForServiceAccounts(saList []service.K8sServiceAccount) []service.MeshService {
	serviceMap := map[service.MeshService]bool{}

	for _, sa := range saList {
		services, err := mc.GetServicesForServiceAccount(sa)
		if err != nil {
			log.Error().Msgf("Error getting services linked to Service Account %s: %v", sa, err)
			continue
		} else {
			for _, s := range services {
				serviceMap[s] = true
			}
		}
	}

	serviceList := []service.MeshService{}
	for k := range serviceMap {
		serviceList = append(serviceList, k)
	}

	return serviceList
}

// GetHostnamesForUpstreamService returns the hostnames over which an upstream service is accessible from a downstream service
// The hostname is the FQDN for the service, and can include ports as well.
// Ex. bookstore.default, bookstore.default:80, bookstore.default.svc, bookstore.default.svc:80 etc.
// TODO: replace GetResolvableHostnamesForUpstreamService with this func once routes refactor is complete (#issue)
func (mc *MeshCatalog) GetHostnamesForUpstreamService(downstream, upstream service.MeshService) ([]string, error) {
	sameNamespace := downstream.Namespace == upstream.Namespace
	// The hostnames for this service are the Kubernetes service DNS names
	hostnames, err := mc.getServiceHostnames(upstream, sameNamespace)
	if err != nil {
		log.Error().Err(err).Msgf("Error getting service hostnames for upstream service %s", upstream)
		return nil, err
	}

	return hostnames, nil
}

// buildTrafficPolicies takes a list of source services, destination services and routes and returns a list of traffic policies for all
//	combinations of the given source and destination services
func (mc *MeshCatalog) buildTrafficPolicies(sourceServices, destServices []service.MeshService, routes []trafficpolicy.HTTPRoute) (policies []*trafficpolicy.TrafficPolicy) {
	for _, sourceService := range sourceServices {
		for _, destService := range destServices {
			if sourceService == destService {
				continue
			}
			routesClusters := []trafficpolicy.RouteWeightedClusters{}
			// When TrafficSplit v1alpha3 is implemented (#705), weighted clusters information should be passed into this function as a parameter
			//	but since we are not implementing TrafficSplit v1alpha3 and only dealing with TrafficTargets for the initial routes refactor(#2034),
			//	we use the default weighted clusters configuration for the given destination service (destService)
			weightedClusters := mapset.NewSet(getDefaultWeightedClusterForService(destService))

			for _, route := range routes {
				routesClusters = append(routesClusters, trafficpolicy.RouteWeightedClusters{
					HTTPRoute:        trafficpolicy.HTTPRoute(route),
					WeightedClusters: weightedClusters,
				})
			}

			hostnames, err := mc.GetHostnamesForUpstreamService(sourceService, destService)
			if err != nil {
				log.Error().Msgf("Err getting resolvable hostnames for source %v and destination %v service : %s", sourceService, destService, err)
				continue
			}
			policies = append(policies, trafficpolicy.NewTrafficPolicy(sourceService, destService, routesClusters, hostnames))
		}
	}
	return policies
}

func isValidTrafficTarget(t *target.TrafficTarget) bool {
	if t.Spec.Rules == nil || len(t.Spec.Rules) == 0 {
		return false
	}
	return true
}
