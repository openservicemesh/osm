package rc

import (
	endpoint2 "github.com/deislabs/smc/pkg/endpoint"
	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/route"
	"github.com/golang/glog"
)

const (
	// RouteConfigurationURI is the string constant of the Route Configuration URI
	RouteConfigurationURI = "type.googleapis.com/envoy.api.v2.RouteConfiguration"
)

//NewRouteConfiguration consrtucts the Envoy construct necessary for TrafficTarget implementation
// todo (snchh) : need to figure out linking to name spaces
// todo (snchh) : trafficPolicies.PolicyRoutePaths.RoutePathMethods not used
func NewRouteConfiguration(trafficPolicies endpoint2.TrafficTargetPolicies) v2.RouteConfiguration {

	rc := v2.RouteConfiguration{
		Name: trafficPolicies.PolicyName,
		VirtualHosts: []*route.VirtualHost{{
			Name:    trafficPolicies.Source,
			Domains: []string{"*"},
			Routes:  []*route.Route{},
		}},
	}

	for _, routePaths := range trafficPolicies.PolicyRoutePaths {
		route := route.Route{
			Match: &route.RouteMatch{
				PathSpecifier: &route.RouteMatch_Prefix{
					Prefix: routePaths.RoutePathRegex,
				},
			},
			Action: &route.Route_Route{
				Route: &route.RouteAction{
					ClusterSpecifier: &route.RouteAction_Cluster{
						Cluster: trafficPolicies.Destination,
					},
				},
			},
		}
		rc.VirtualHosts[0].Routes = append(rc.VirtualHosts[0].Routes, &route)
	}

	glog.V(7).Infof("[RDS] Constructed RouteConfiguration: %+v", rc)
	return rc
}
