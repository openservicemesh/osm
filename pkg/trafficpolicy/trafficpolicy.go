package trafficpolicy

import (
	"reflect"

	set "github.com/deckarep/golang-set"

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
