package trafficpolicy

import (
	"fmt"
	"strings"

	policyv1alpha1 "github.com/openservicemesh/osm/pkg/apis/policy/v1alpha1"
)

// EgressTrafficPolicy is the type used to represent the different egress traffic policy configurations
// applicable to a client of Egress destinations.
type EgressTrafficPolicy struct {
	// TrafficMatches defines the list of traffic matches for matching Egress traffic.
	// The matches specified are used to match outbound traffic as Egress traffic, and
	// subject matching traffic to Egress traffic policies.
	TrafficMatches []*TrafficMatch

	// HTTPRouteConfigsPerPort defines the Egress HTTP route configurations per port.
	// Egress HTTP routes are grouped based on their port to avoid route conflicts that
	// can arise when the same host headers are to be routed differently based on the
	// port specified in an egress policy.
	HTTPRouteConfigsPerPort map[int][]*EgressHTTPRouteConfig

	// ClustersConfigs defines the list of Egress cluster configurations.
	// The specified config is used to program external clusters corresponding to
	// the external endpoints defined in an Egress policy.
	ClustersConfigs []*EgressClusterConfig
}

// EgressClusterConfig is the type used to represent an external cluster corresponding to a
// destination specified in an Egress policy.
type EgressClusterConfig struct {
	// Name defines the name of the external cluster
	Name string

	// Host defines the DNS resolvabe hostname for the external cluster.
	// If specified, the cluster's address will be resolved using DNS.
	// HTTP based clusters will set the Host attribute.
	// If unspecified, the cluster's address will be resolved to its original
	// destination in the request prior to being redirected by iptables.
	// TCP based clusters will not set the Host attribute.
	// +optional
	Host string

	// Port defines the port number of the external cluster's endpoint
	Port int

	// UpstreamTrafficSetting is the traffic setting for the upstream cluster
	UpstreamTrafficSetting *policyv1alpha1.UpstreamTrafficSetting
}

// EgressHTTPRouteConfig is the type used to represent an HTTP route configuration along with associated routing rules
type EgressHTTPRouteConfig struct {
	// Name defines the name of the Egress HTTP route configuration
	Name string

	// Hostnames defines the list of hostnames corresponding to the Egress HTTP route configuration.
	// The Hostnames match against the :host header in the HTTP request and subject matching requests
	// to the routing rules defined by `RoutingRules`.
	Hostnames []string

	// RoutingRules defines the list of routes for the Egress HTTP route configuration, and corresponding
	// rules to be applied to those routes.
	RoutingRules []*EgressHTTPRoutingRule
}

// EgressHTTPRoutingRule is the type used to represent an Egress HTTP routing rule with its route and associated permissions
type EgressHTTPRoutingRule struct {
	// Route defines the HTTP route match and its associated cluster.
	Route RouteWeightedClusters

	// AllowedDestinationIPRanges defines the destination IP ranges allowed for the `Route` defined in the routing rule.
	AllowedDestinationIPRanges []string
}

// GetEgressTrafficMatchName returns the name for the TrafficMatch object based on
// its port and protocol
func GetEgressTrafficMatchName(port int, protocol string) string {
	protocol = strings.ToLower(protocol)
	return fmt.Sprintf("egress-%s.%d", protocol, port)
}
