package trafficpolicy

import (
	"reflect"
	"sort"

	mapset "github.com/deckarep/golang-set"
	"github.com/pkg/errors"

	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/service"
)

// WildCardRouteMatch represents a wildcard HTTP route match condition
var WildCardRouteMatch HTTPRouteMatch = HTTPRouteMatch{
	Path:          constants.RegexMatchAll,
	PathMatchType: PathMatchRegex,
	Methods:       []string{constants.WildcardHTTPMethod},
}

// NewRouteWeightedCluster takes a route and weighted cluster and returns a *RouteWeightedCluster
func NewRouteWeightedCluster(route HTTPRouteMatch, weightedClusters []service.WeightedCluster) *RouteWeightedClusters {
	weightedClusterSet := mapset.NewSet()
	for _, wc := range weightedClusters {
		weightedClusterSet.Add(wc)
	}

	return &RouteWeightedClusters{
		HTTPRouteMatch:   route,
		WeightedClusters: weightedClusterSet,
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
			AllowedServiceAccounts: mapset.NewSet(allowedServiceAccount),
		})
	}
}

// AddRoute adds a route to an OutboundTrafficPolicy given an HTTP route match and weighted cluster. If a Route with the given HTTP route match
//	already exists, an error will be returned. If a Route with the given HTTP route match does not exist,
//	a Route with the given HTTP route match and weighted clusters will be added to the Routes on the OutboundTrafficPolicy
func (out *OutboundTrafficPolicy) AddRoute(httpRouteMatch HTTPRouteMatch, weightedClusters ...service.WeightedCluster) error {
	wc := mapset.NewSet()
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
// allowPartialHostnamesMatch when set to true merges inbound policies by partially comparing (subset of one another) the hostnames of the original traffic policy to the latest traffic policy
// A partial match on hostnames should be allowed for the following scenarios :
// 1. when an ingress policy is being merged with other ingress traffic policies or
// 2. when a policy having its hostnames from a host header needs to be merged with other inbound traffic policies
// in either of these cases the will be only a single hostname and there is a possibility that this hostname is part of an existing traffic policy
// hence the rules need to be merged
func MergeInboundPolicies(allowPartialHostnamesMatch bool, original []*InboundTrafficPolicy, latest ...*InboundTrafficPolicy) []*InboundTrafficPolicy {
	for _, l := range latest {
		foundHostnames := false
		for _, or := range original {
			if !allowPartialHostnamesMatch {
				if reflect.DeepEqual(or.Hostnames, l.Hostnames) {
					foundHostnames = true
					or.Rules = mergeRules(or.Rules, l.Rules)
				}
			} else {
				// If l.Hostnames is a subset of or.Hostnames or vice versa then we need to get a union of the two
				if hostsUnion := slicesUnionIfSubset(or.Hostnames, l.Hostnames); len(hostsUnion) > 0 {
					or.Hostnames = hostsUnion
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
// allowPartialHostnamesMatch when set to true merges outbound policies by partially comparing (subset of one another) the hostnames of the original traffic policy to the latest traffic policy
// Two outbound traffic policies can be merged by comparing their hostnames either partially or completely. A partial match on hostnames should be allowed for the following scenarios :
// when a policy having its hostnames from a host header needs to be merged with other outbound policies
// This is because there will be a single hostname (from the host header) and there is a possibility that this hostname is part of an existing traffic policy
// hence the rules need to be merged
func MergeOutboundPolicies(allowPartialHostnamesMatch bool, original []*OutboundTrafficPolicy, latest ...*OutboundTrafficPolicy) []*OutboundTrafficPolicy {
	for _, l := range latest {
		foundHostnames := false
		for _, or := range original {
			if !allowPartialHostnamesMatch {
				if reflect.DeepEqual(or.Hostnames, l.Hostnames) {
					foundHostnames = true
					mergedRoutes := mergeRoutesWeightedClusters(or.Routes, l.Routes)
					or.Routes = mergedRoutes
				}
			} else {
				// If l.Hostnames is a subset of or.Hostnames or vice versa then we need to get a union of the two
				if hostsUnion := slicesUnionIfSubset(or.Hostnames, l.Hostnames); len(hostsUnion) > 0 {
					or.Hostnames = hostsUnion
					foundHostnames = true
					mergedRoutes := mergeRoutesWeightedClusters(or.Routes, l.Routes)
					or.Routes = mergedRoutes
				}
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

// slicesUnionIfSubset returns the union of the two slices if either slices is a subset of the other
func slicesUnionIfSubset(first, second []string) []string {
	areSubsets := false
	unionSlice := []string{}
	firstIntf := convertToInterface(first)
	secondIntf := convertToInterface(second)

	firstSet := mapset.NewSetFromSlice(firstIntf)
	secondSet := mapset.NewSetFromSlice(secondIntf)

	if firstSet.IsSubset(secondSet) || secondSet.IsSubset(firstSet) {
		areSubsets = true
	}

	if areSubsets {
		union := firstSet.Union(secondSet)
		for intf := range union.Iter() {
			unionSlice = append(unionSlice, intf.(string))
		}
		sort.Strings(unionSlice)
		return unionSlice
	}
	return unionSlice
}

func convertToInterface(slice []string) []interface{} {
	sliceInterface := make([]interface{}, len(slice))
	for i := range slice {
		sliceInterface[i] = slice[i]
	}
	return sliceInterface
}
