package catalog

import (
	"fmt"

	mapset "github.com/deckarep/golang-set"
	"github.com/pkg/errors"
	target "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/access/v1alpha2"

	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/kubernetes"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/trafficpolicy"
)

const (
	//HTTPTraffic specifies HTTP Traffic Policy
	HTTPTraffic = "HTTPRouteGroup"
)

var errNoTrafficSpecsFound = errors.New("No Traffic Specs found")

var allowAllRoute trafficpolicy.HTTPRoute = trafficpolicy.HTTPRoute{
	PathRegex: constants.RegexMatchAll,
	Methods:   []string{constants.WildcardHTTPMethod},
}

// ListTrafficPolicies returns a list of traffic policies associated with the given service account
func (mc *MeshCatalog) ListTrafficPoliciesForService(sa service.K8sServiceAccount) ([]*trafficpolicy.TrafficPolicy, []*trafficpolicy.TrafficPolicy, error) {

	if mc.configurator.IsPermissiveTrafficPolicyMode() {
		// Build traffic policies from service discovery for allow-all policy
		return mc.buildAllowAllTrafficPolicies(sa)
	}

	inbound, outbound, _ := mc.listTrafficPoliciesFromTrafficTargets(sa)

	/*
		TODO: outboundFromSplits := mc.ListTrafficPoliciesFromTrafficSplits()
		for _, out := range outboundFromSplits {
			outbound = append(outbound, out)
		}
		outbound = consolidate(outbound)

	*/

	//TODO find any ingress resources and create inboundPolicies based on the ingress resources
	//	and then consolidate inbound with the routes from ingress
	//ingress has host: hostname.namespace
	// existing trafficpolicy { hostname.namespace, hostname, hostname}
	//ingressPolicies, _ := mc.GetIngressTrafficPolicies(sa)
	//inbound = consolidateIngressPolicies(ingressPolicies, inbound)
	return inbound, outbound, nil

}

// foundHostMatch takes two string slices and returns a string slice that contains all matching
//	strings between the two slices and a boolean indicating whether any matching strings were found
func foundHostMatch(hostnames1, hostnames2 []string) ([]string, bool) {
	matches := []string{}
	found := false
	for _, h1 := range hostnames1 {
		for _, h2 := range hostnames2 {
			if h1 == h2 {
				found = true
				matches = append(matches, h1)
			}
		}
	}
	return matches, found
}

// removeHostnames takes a list of the original hostnames and the a list of the hostnames to be removed
//	and returns a list of hostnames without the hostnames in the remove list
func removeHostnames(originalList, removeList []string) []string {
	updatedList := []string{}
	for _, original := range originalList {
		found := false
		for _, remove := range removeList {
			if original == remove {
				found = true
				break
			}
		}
		if found {
			continue
		} else {
			// since original is not in the removeList append it to the updatedList
			updatedList = append(updatedList, original)
		}

	}

	return updatedList
}

// consolidateIngressPolicies takes traffic policies related to ingress resources and traffic policies related to SMI TrafficTargets
//	and SMI TrafficSplits and returns a list of traffic policies where the domains do not overlap but the routes for the domains
//	are preserved
func consolidateIngressPolicies(ingressPolicies, inboundPolicies []*trafficpolicy.TrafficPolicy) []*trafficpolicy.TrafficPolicy {
	final := []*trafficpolicy.TrafficPolicy{}
	for _, ingress := range ingressPolicies {
		for _, inbound := range inboundPolicies {
			matches, found := foundHostMatch(ingress.Hostnames, inbound.Hostnames)
			if found {
				inbound.Hostnames = removeHostnames(inbound.Hostnames, matches)
				for _, route := range inbound.HTTPRoutesClusters {
					ingress.HTTPRoutesClusters = append(ingress.HTTPRoutesClusters, route)
				}
			}
			final = append(final, inbound)
		}
		final = append(final, ingress)
	}

	return final
}

// consolidatePoliciesByDestination consolidates the given traffic policies by destination service and returns a list of
//	traffic policies where there is a policy for each unique destination service and all HTTPRouteWeightedClusters
// 	for that destination service are merged into a single list of HTTPRouteWeightedClusters
func consolidatePoliciesByDestination(policies []*trafficpolicy.TrafficPolicy) []*trafficpolicy.TrafficPolicy {
	policyKeys := make(map[string]*trafficpolicy.TrafficPolicy)
	uniquePolicies := []*trafficpolicy.TrafficPolicy{}
	for _, policy := range policies {
		if foundPolicy, found := policyKeys[policy.Name]; !found {
			policyKeys[policy.Name] = policy
			uniquePolicies = append(uniquePolicies, policy)
		} else {
			// if a policy with the name already exists, merge the HTTPRoutesClusters slices
			for _, r := range policy.HTTPRoutesClusters {
				foundPolicy.HTTPRoutesClusters = append(foundPolicy.HTTPRoutesClusters, r)
			}
		}
	}
	return uniquePolicies
}

// listTrafficPoliciesFromTrafficTargets loops through all SMI Traffic Target resources and returns inbound traffic policies and outbound policies
//		based on when the given service account matches a destination or source in the Traffic Target resource
func (mc *MeshCatalog) listTrafficPoliciesFromTrafficTargets(sa service.K8sServiceAccount) ([]*trafficpolicy.TrafficPolicy, []*trafficpolicy.TrafficPolicy, error) {

	inboundPolicies := []*trafficpolicy.TrafficPolicy{}
	outboundPolicies := []*trafficpolicy.TrafficPolicy{}
	for _, t := range mc.meshSpec.ListTrafficTargets() { // loop through all traffic targets
		if !validTrafficTarget(t) {
			continue
		}

		if t.Spec.Destination.Name == sa.Name { // found inbound
			routes, err := mc.HTTPRoutesFromRules(t.Spec.Rules, t.Namespace)
			if err != nil {
				log.Error().Msgf("Err finding route matches from TrafficTarget %s in namespace %s: %v", t.Name, t.Namespace, err)
				break
			}

			destServices, err := mc.GetServicesForServiceAccount(sa)
			if err != nil {
				log.Error().Msgf("Err finding Services for Service Account %#v: %v", sa, err)
				return nil, nil, err
			}

			sourceServices := mc.GetServicesForServiceAccounts(serviceAccountsForSources(t.Spec.Sources))

			policies := mc.buildTrafficPolicies(sourceServices, destServices, routes)
			for _, policy := range policies {
				inboundPolicies = append(inboundPolicies, policy)
			}
			continue
		}

		for _, source := range t.Spec.Sources {
			if source.Name == sa.Name { // found outbound

				destServices, err := mc.GetServicesForServiceAccount(service.K8sServiceAccount{
					Name:      t.Spec.Destination.Name,
					Namespace: t.Namespace,
				})
				if err != nil {
					log.Error().Msgf("No Services found matching Service Account %s in Namespace %s", t.Spec.Destination.Name, t.Namespace)
					continue
				}

				sourceServices, err := mc.GetServicesForServiceAccount(sa)
				if err != nil {
					log.Error().Msgf("Err finding Services for Service Account %#v: %v", sa, err)
					return nil, nil, err
				}

				outPolicies := mc.buildTrafficPolicies(sourceServices, destServices, []trafficpolicy.HTTPRoute{allowAllRoute})
				for _, policy := range outPolicies {
					outboundPolicies = append(outboundPolicies, policy)
				}
				break
			}
		}

	}
	return consolidatePoliciesByDestination(inboundPolicies), consolidatePoliciesByDestination(outboundPolicies), nil
}

func (mc *MeshCatalog) HTTPRoutesFromRules(rules []target.TrafficTargetRule, namespace string) ([]trafficpolicy.HTTPRoute, error) {
	routes := []trafficpolicy.HTTPRoute{}

	specMatchRoute, err := mc.getHTTPPathsPerRoute() // returns map[spec_name]map[match_name]trafficpolicy.HTTPRoute
	if err != nil {
		return nil, err
	}

	if len(specMatchRoute) == 0 {

		return routes, errNoTrafficSpecsFound
	}

	for _, rule := range rules {
		trafficSpecName := mc.getTrafficSpecName("HTTPRouteGroup", namespace, rule.Name)
		for _, match := range rule.Matches {
			matchedRoute, found := specMatchRoute[trafficSpecName][trafficpolicy.TrafficSpecMatchName(match)]
			if found {
				routes = append(routes, matchedRoute)
			} else {
				// TODO handle match not found
			}
		}

	}

	return routes, nil
}

func (mc *MeshCatalog) buildTrafficPolicies(sourceServices, destServices []service.MeshService, routes []trafficpolicy.HTTPRoute) (policies []*trafficpolicy.TrafficPolicy) {

	for _, sourceService := range sourceServices {
		for _, destService := range destServices {
			if sourceService == destService {
				continue
			}
			routesClusters := []trafficpolicy.RouteWeightedClusters{}
			weightedClusters := mapset.NewSet(getDefaultWeightedClusterForService(destService))

			for _, route := range routes {
				routesClusters = append(routesClusters, trafficpolicy.RouteWeightedClusters{
					HTTPRoute:        trafficpolicy.HTTPRoute(route),
					WeightedClusters: weightedClusters,
					// TODO on inbound do we need to also program the weightedclusters?
				})
			}

			hostnames, err := mc.GetResolvableHostnamesForUpstreamService(sourceService, destService)
			if err != nil {
				log.Error().Msgf("Err getting resolvable hostnames for source service %v and destination service %v: %s", sourceService, destService, err)
				continue
			}

			policies = append(policies, trafficpolicy.NewTrafficPolicy(sourceService, destService, routesClusters, hostnames))
		}

	}
	return policies

}

// This function returns the list of connected services.
// This is a bimodal function:
//   - it could list services that are allowed to connect to the given service (inbound)
//   - it could list services that the given service can connect to (outbound)
func (mc *MeshCatalog) getAllowedDirectionalServices(sa service.K8sServiceAccount, dir trafficDirection) ([]service.MeshService, error) {
	//allTrafficPolicies, err := mc.ListTrafficPolicies(svc)
	inboundPolicies, outboundPolicies, err := mc.ListTrafficPoliciesForService(sa) // TODO place with listTrafficPoliciesFromTrafficTargets

	if err != nil {
		log.Error().Err(err).Msg("Failed listing traffic routes")
		return nil, err
	}

	services, _ := mc.GetServicesForServiceAccount(sa)

	allowedServicesSet := mapset.NewSet()

	if dir == inbound {
		for _, policy := range inboundPolicies {
			// we are looking for services that can connect to the given service
			for _, svc := range services {
				if policy.Destination.Equals(svc) {
					allowedServicesSet.Add(policy.Source)
					break
				}
			}
		}
	}

	if dir == outbound {
		for _, policy := range outboundPolicies {
			for _, svc := range services {
				if policy.Source.Equals(svc) {
					allowedServicesSet.Add(policy.Destination)
					break
				}
			}
		}
	}

	// Convert the set of interfaces to a list of namespaced services
	var allowedServices []service.MeshService
	for svc := range allowedServicesSet.Iter() {
		allowedServices = append(allowedServices, svc.(service.MeshService))
	}

	msg := map[trafficDirection]string{
		inbound:  "Allowed inbound services for destination %q: %+v",
		outbound: "Allowed outbound services from source %q: %+v",
	}[dir]

	log.Trace().Msgf(msg, sa, allowedServices)

	return allowedServices, nil
}

// ListAllowedInboundServices lists the inbound services allowed to connect to the given service.
func (mc *MeshCatalog) ListAllowedInboundServices(sa service.K8sServiceAccount) ([]service.MeshService, error) {
	allowedInboundServices, err := mc.getAllowedDirectionalServices(sa, inbound)

	return allowedInboundServices, err
}

// ListAllowedOutboundServices lists the services the given service is allowed outbound connections to.
func (mc *MeshCatalog) ListAllowedOutboundServices(sa service.K8sServiceAccount) ([]service.MeshService, error) {
	return mc.getAllowedDirectionalServices(sa, outbound)
}

// GetResolvableHostnamesForUpstreamService returns the hostnames over which an upstream service is accessible from a downstream service
// The hostname is the FQDN for the service, and can include ports as well.
// Ex. bookstore.default, bookstore.default:80, bookstore.default.svc, bookstore.default.svc:80 etc.
func (mc *MeshCatalog) GetResolvableHostnamesForUpstreamService(downstream, upstream service.MeshService) ([]string, error) {
	sameNamespace := downstream.Namespace == upstream.Namespace
	// The hostnames for this service are the Kubernetes service DNS names.
	hostnames, err := mc.getServiceHostnames(upstream, sameNamespace)
	if err != nil {
		log.Error().Err(err).Msgf("Error getting service hostnames for upstream service %s", upstream)
		return nil, err
	}

	return hostnames, nil
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

func (mc *MeshCatalog) buildAllowAllTrafficPolicies(sa service.K8sServiceAccount) (inbound []*trafficpolicy.TrafficPolicy, outbound []*trafficpolicy.TrafficPolicy, err error) {
	services, err := mc.GetServicesForServiceAccount(sa)
	if err != nil {
		return inbound, outbound, err
	}
	allServices := kubernetesServicesToMeshServices(mc.kubeController.ListServices())
	inbound = mc.buildTrafficPolicies(allServices, services, []trafficpolicy.HTTPRoute{allowAllRoute})
	outbound = mc.buildTrafficPolicies(services, allServices, []trafficpolicy.HTTPRoute{allowAllRoute})

	return consolidatePoliciesByDestination(inbound), consolidatePoliciesByDestination(outbound), err
}

func consolidatePolicies(policies []*trafficpolicy.TrafficPolicy) []*trafficpolicy.TrafficPolicy {
	policyKeys := make(map[string]*trafficpolicy.TrafficPolicy)
	uniquePolicies := []*trafficpolicy.TrafficPolicy{}
	for _, policy := range policies {
		if foundPolicy, found := policyKeys[policy.Name]; !found {
			policyKeys[policy.Name] = policy
			uniquePolicies = append(uniquePolicies, policy)
		} else {
			// if a policy with the name already exists, merge the HTTPRoutesClusters slices
			for _, r := range policy.HTTPRoutesClusters {
				foundPolicy.HTTPRoutesClusters = append(foundPolicy.HTTPRoutesClusters, r)
			}
		}
	}
	return uniquePolicies

}

func getDefaultWeightedClusterForService(meshService service.MeshService) service.WeightedCluster {
	log.Debug().Msgf("In default weighted cluster for service %v: service.ClusterName is %v\nmeshService.String() is %v", meshService, service.ClusterName(meshService.String()), meshService.String())

	return service.WeightedCluster{
		ClusterName: service.ClusterName(meshService.String()),
		Weight:      constants.ClusterWeightAcceptAll,
	}
}

func serviceAccountsForSources(sources []target.IdentityBindingSubject) []service.K8sServiceAccount {
	serviceAccounts := []service.K8sServiceAccount{}
	for _, source := range sources {
		serviceAccounts = append(serviceAccounts, service.K8sServiceAccount{
			Name:      source.Name,
			Namespace: source.Namespace,
		})
	}
	return serviceAccounts
}

func validTrafficTarget(t *target.TrafficTarget) bool {
	if t.Spec.Rules == nil || len(t.Spec.Rules) == 0 {
		log.Error().Msgf("Skipping TrafficTarget %s/%s is invalid (has no rules)\n", t.Namespace, t.Name)
		return false
	}
	return true
}
