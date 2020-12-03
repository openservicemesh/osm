package trafficpolicy

import (
	"reflect"

	set "github.com/deckarep/golang-set"
	"github.com/rs/zerolog/log"

	"github.com/openservicemesh/osm/pkg/service"
)

// AddRuleForRoute adds a Rule to an InboundTrafficPolicy based on the given HTTP route match, weighted cluster, and service account parameters.
//	If a Rule for the given HTTP route match exists, it will add the given service account to the Rule.
//	If the the given route match is not already associated with a Rule, it will create a Rule for the given route and service account.
func (in *InboundTrafficPolicy) AddRuleForRoute(route RouteWeightedClusters, sa service.K8sServiceAccount) {
	routeExists := false
	for _, rule := range in.Rules {
		if reflect.DeepEqual(rule.Route, route) {
			routeExists = true
			rule.ServiceAccounts.Add(sa)
			break
		}
	}
	if !routeExists {
		in.Rules = append(in.Rules, &Rule{
			Route:           route,
			ServiceAccounts: set.NewSet(sa),
		})
	}
}

// AddRoute adds a route to an OutboundTrafficPolicy given an HTTP route match and weighted cluster. If a Route with the given HTTP route match
//	already exists, no change will occur to the Routes on the OutboundTrafficPolicy. If a Route with the given HTTP route match does not exist,
//	a Route with the given HTTP route match and weighted clusters will be added to the Routes on the OutboundTrafficPolicy
func (out *OutboundTrafficPolicy) AddRoute(httpRouteMatch HTTPRouteMatch, weightedCluster service.WeightedCluster) {
	routeExists := false
	for _, existingRoute := range out.Routes {
		if reflect.DeepEqual(existingRoute.HTTPRouteMatch, httpRouteMatch) {
			routeExists = true
			log.Debug().Msgf("Ignoring as route %v already exists for %s", httpRouteMatch, out.Name)
			return
		}
	}

	if !routeExists {
		out.Routes = append(out.Routes, &RouteWeightedClusters{
			HTTPRouteMatch:   httpRouteMatch,
			WeightedClusters: set.NewSet(weightedCluster),
		})
	}
}

// TotalClustersWeight returns total weight of the WeightedClusters on a RouteWeightedClusters
func (rwc *RouteWeightedClusters) TotalClustersWeight() int {
	var totalWeight int
	if rwc.WeightedClusters.Cardinality() > 0 {
		for clusterInterface := range rwc.WeightedClusters.Iter() { // iterate
			cluster := clusterInterface.(service.WeightedCluster)
			totalWeight += cluster.Weight
		}
	}
	return totalWeight
}

// MergeInboundPolicies merges InboundTrafficPolicy objects to a slice of InboundTrafficPolicy so that there is one InboundTrafficPolicy for
//	a set of hostnames and all Rules are merged into a single InboundTrafficPolicy
func MergeInboundPolicies(inbound []*InboundTrafficPolicy, policies ...*InboundTrafficPolicy) []*InboundTrafficPolicy {
	for _, p := range policies {
		foundHostnames := false
		for _, in := range inbound {
			if reflect.DeepEqual(in.Hostnames, p.Hostnames) {
				foundHostnames = true
				rules := mergeRules(in.Rules, p.Rules)
				in.Rules = rules
			}
		}
		if !foundHostnames {
			inbound = append(inbound, p)
		}
	}
	return inbound
}

// MergeOutboundPolicies merges OutboundTrafficPolicy objects to a slice of ObjectTrafficPolicy so that there is one OutboundTrafficPolicy
//	for a set of Hostnames and all Routes associated with the set of Hostnames are merged into a single OutboundTrafficPolicy
func MergeOutboundPolicies(outbound []*OutboundTrafficPolicy, policies ...*OutboundTrafficPolicy) []*OutboundTrafficPolicy {
	for _, p := range policies {
		foundHostnames := false
		for _, out := range outbound {
			if reflect.DeepEqual(out.Hostnames, p.Hostnames) {
				foundHostnames = true
				out.Routes = mergeRoutes(out.Routes, p.Routes)
			}
		}
		if !foundHostnames {
			outbound = append(outbound, p)
		}
	}
	return outbound
}

func mergeRules(originalRules, latestRules []*Rule) []*Rule {
	for _, latest := range latestRules {
		foundRoute := false
		for _, original := range originalRules {
			if reflect.DeepEqual(latest.Route, original.Route) {
				foundRoute = true
				original.ServiceAccounts.Add(latest.ServiceAccounts)
			}
		}
		if !foundRoute {
			originalRules = append(originalRules, latest)
		}
	}
	return originalRules
}

func mergeRoutes(originalRoutes, latestRoutes []*RouteWeightedClusters) []*RouteWeightedClusters {
	// find if latest route is in original
	for _, latest := range latestRoutes {
		foundRoute := false
		for _, original := range originalRoutes {
			if reflect.DeepEqual(original.HTTPRouteMatch, latest.HTTPRouteMatch) {
				foundRoute = true
				//TODO take the latest route
				//TODO add debug line
				continue
			}
		}
		if !foundRoute {
			originalRoutes = append(originalRoutes, latest)
		}
	}
	return originalRoutes
}
