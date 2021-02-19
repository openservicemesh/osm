package trafficpolicy

import (
	"reflect"

	set "github.com/deckarep/golang-set"
	"github.com/pkg/errors"

	"github.com/openservicemesh/osm/pkg/service"
)

// NewRouteWeightedCluster takes a route and weighted cluster and returns a *RouteWeightedCluster
func NewRouteWeightedCluster(route HTTPRouteMatch, weightedCluster service.WeightedCluster) *RouteWeightedClusters {
	return &RouteWeightedClusters{
		HTTPRouteMatch:   route,
		WeightedClusters: set.NewSet(weightedCluster),
	}
}

// NewInboundTrafficPolicy takes a name and list of hostnames and returns an *InboundTrafficPolicy
func NewInboundTrafficPolicy(name string, hostnames []string) *InboundTrafficPolicy {
	return &InboundTrafficPolicy{
		Name:      name,
		Hostnames: hostnames,
	}
}

// NewOutboundTrafficPolicy takes a name and list of hostnames and returns an *OutboundTrafficPolicy
func NewOutboundTrafficPolicy(name string, hostnames []string) *OutboundTrafficPolicy {
	return &OutboundTrafficPolicy{
		Name:      name,
		Hostnames: hostnames,
	}
}

// TotalClustersWeight returns total weight of the WeightedClusters in RouteWeightedClusters
func (rwc *RouteWeightedClusters) TotalClustersWeight() int {
	var totalWeight int
	for clusterInterface := range rwc.WeightedClusters.Iter() { // iterate
		cluster := clusterInterface.(service.WeightedCluster)
		totalWeight += cluster.Weight
	}
	return totalWeight
}

// AddRule adds a Rule to an InboundTrafficPolicy based on the given HTTP route match, weighted cluster, and allowed service account
//	parameters. If a Rule for the given HTTP route match exists, it will add the given service account to the Rule. If the the given route
//	match is not already associated with a Rule, it will create a Rule for the given route and service account.
func (in *InboundTrafficPolicy) AddRule(route RouteWeightedClusters, allowedServiceAccount service.K8sServiceAccount) {
	routeExists := false
	for _, rule := range in.Rules {
		if reflect.DeepEqual(rule.Route, route) {
			routeExists = true
			rule.AllowedServiceAccounts.Add(allowedServiceAccount)
			break
		}
	}
	if !routeExists {
		in.Rules = append(in.Rules, &Rule{
			Route:                  route,
			AllowedServiceAccounts: set.NewSet(allowedServiceAccount),
		})
	}
}

// AddRoute adds a route to an OutboundTrafficPolicy given an HTTP route match and weighted cluster. If a Route with the given HTTP route match
//	already exists, an error will be returned. If a Route with the given HTTP route match does not exist,
//	a Route with the given HTTP route match and weighted clusters will be added to the Routes on the OutboundTrafficPolicy
func (out *OutboundTrafficPolicy) AddRoute(httpRouteMatch HTTPRouteMatch, weightedClusters ...service.WeightedCluster) error {
	wc := set.NewSet()
	for _, c := range weightedClusters {
		wc.Add(c)
	}

	for _, existingRoute := range out.Routes {
		if reflect.DeepEqual(existingRoute.HTTPRouteMatch, httpRouteMatch) {
			if existingRoute.WeightedClusters.Equal(wc) {
				return nil
			}
			return errors.Errorf("Route for HTTP Route Match: %v already exists: %v for outbound traffic policy: %s", existingRoute.HTTPRouteMatch, existingRoute, out.Name)
		}
	}

	out.Routes = append(out.Routes, &RouteWeightedClusters{
		HTTPRouteMatch:   httpRouteMatch,
		WeightedClusters: wc,
	})
	return nil
}

// MergeInboundPolicies merges latest InboundTrafficPolicies into a slice of InboundTrafficPolicies that already exists (original)
// allowPartialHostnamesMatch is set to true when we intend to merge ingress policies with traffic policies
func MergeInboundPolicies(allowPartialHostnamesMatch bool, original []*InboundTrafficPolicy, latest ...*InboundTrafficPolicy) []*InboundTrafficPolicy {
	for _, l := range latest {
		foundHostnames := false
		for _, or := range original {
			if !allowPartialHostnamesMatch {
				// For an traffic target inbound policy the hostnames list should fully intersect
				// to merge the rules
				if reflect.DeepEqual(or.Hostnames, l.Hostnames) {
					foundHostnames = true
					or.Rules = mergeRules(or.Rules, l.Rules)
				}
			} else {
				// When an inbound traffic policy is being merged with an ingress traffic policy the hostnames is not the entire comprehensive list of kubernetes service names
				// and will just be a subset to merge the rules
				if subset(or.Hostnames, l.Hostnames) {
					foundHostnames = true
					or.Rules = mergeRules(or.Rules, l.Rules)
				}
			}
		}
		if !foundHostnames {
			original = append(original, l)
		}
	}
	return original
}

// MergeOutboundPolicies merges two slices of *OutboundTrafficPolicies so that there is only one traffic policy for a given set of a hostnames
func MergeOutboundPolicies(original []*OutboundTrafficPolicy, latest ...*OutboundTrafficPolicy) []*OutboundTrafficPolicy {
	for _, l := range latest {
		foundHostnames := false
		for _, or := range original {
			if reflect.DeepEqual(or.Hostnames, l.Hostnames) {
				foundHostnames = true
				mergedRoutes := mergeRoutesWeightedClusters(or.Routes, l.Routes)
				or.Routes = mergedRoutes
			}
		}
		if !foundHostnames {
			original = append(original, l)
		}
	}
	return original
}

// mergeRules merges the give slices of rules such that there is one Rule for a Route with all allowed service accounts listed in the
//	returned slice of rules
func mergeRules(originalRules, latestRules []*Rule) []*Rule {
	for _, latest := range latestRules {
		foundRoute := false
		for _, original := range originalRules {
			if reflect.DeepEqual(latest.Route, original.Route) {
				foundRoute = true
				original.AllowedServiceAccounts = original.AllowedServiceAccounts.Union(latest.AllowedServiceAccounts)
				break
			}
		}
		if !foundRoute {
			originalRules = append(originalRules, latest)
		}
	}
	return originalRules
}

// mergeRoutesWeightedClusters merges two slices of RouteWeightedClusters and returns a slice where there is one RouteWeightedCluster
//	for any HTTPRouteMatch. Where there is an overlap in HTTPRouteMatch between the originalRoutes and latestRoutes, the WeightedClusters
//  specified in the latestRoutes will be kept since there can only be one set of WeightedClusters per HTTPRouteMatch.
func mergeRoutesWeightedClusters(originalRoutes, latestRoutes []*RouteWeightedClusters) []*RouteWeightedClusters {
	for _, latest := range latestRoutes {
		foundRoute := false
		for _, original := range originalRoutes {
			if reflect.DeepEqual(original.HTTPRouteMatch, latest.HTTPRouteMatch) {
				foundRoute = true
				if !reflect.DeepEqual(original.WeightedClusters, latest.WeightedClusters) {
					original.WeightedClusters = latest.WeightedClusters
				}
				continue
			}
		}
		if !foundRoute {
			originalRoutes = append(originalRoutes, latest)
		}
	}
	return originalRoutes
}

// subset returns true if the second array is completely contained in the first array
func subset(first, second []string) bool {
	set := make(map[string]bool)
	for _, value := range first {
		set[value] = true
	}

	for _, value := range second {
		if _, found := set[value]; !found {
			return false
		}
	}

	return true
}
