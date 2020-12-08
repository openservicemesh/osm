package trafficpolicy

import (
	"reflect"

	set "github.com/deckarep/golang-set"
	"github.com/pkg/errors"

	"github.com/openservicemesh/osm/pkg/service"
)

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
	routeExists := false
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

	if !routeExists {
		out.Routes = append(out.Routes, &RouteWeightedClusters{
			HTTPRouteMatch:   httpRouteMatch,
			WeightedClusters: wc,
		})
	}
	return nil
}
