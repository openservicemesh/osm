package injector

import (
	"fmt"

	"github.com/openservicemesh/osm/pkg/constants"
)

// iptablesRedirectionChains is the list of iptables chains created for traffic redirection via the proxy sidecar
var iptablesRedirectionChains = []string{
	// Chain to intercept inbound traffic
	"iptables -t nat -N PROXY_INBOUND",

	// Chain to redirect inbound traffic to the proxy
	"iptables -t nat -N PROXY_IN_REDIRECT",

	// Chain to intercept outbound traffic
	"iptables -t nat -N PROXY_OUTPUT",

	// Chain to redirect outbound traffic to the proxy
	"iptables -t nat -N PROXY_REDIRECT",
}

// iptablesOutboundStaticRules is the list of iptables rules related to outbound traffic interception and redirection
var iptablesOutboundStaticRules = []string{
	// Redirects outbound TCP traffic hitting PROXY_REDIRECT chain to Envoy's outbound listener port
	fmt.Sprintf("iptables -t nat -A PROXY_REDIRECT -p tcp -j REDIRECT --to-port %d", constants.EnvoyOutboundListenerPort),

	// Traffic to the Proxy Admin port flows to the Proxy -- not redirected
	fmt.Sprintf("iptables -t nat -A PROXY_REDIRECT -p tcp --dport %d -j ACCEPT", constants.EnvoyAdminPort),

	// For outbound TCP traffic jump from OUTPUT chain to PROXY_OUTPUT chain
	"iptables -t nat -A OUTPUT -p tcp -j PROXY_OUTPUT",

	// TODO(#1266): Redirect app back calls to itself using PROXY_UID

	// Don't redirect Envoy traffic back to itself for non-loopback traffic
	fmt.Sprintf("iptables -t nat -A PROXY_OUTPUT -m owner --uid-owner %d -j RETURN", constants.EnvoyUID),

	// Skip localhost traffic, doesn't need to be routed via the proxy
	"iptables -t nat -A PROXY_OUTPUT -d 127.0.0.1/32 -j RETURN",

	// Redirect remaining outbound traffic to Envoy
	"iptables -t nat -A PROXY_OUTPUT -j PROXY_REDIRECT",
}

// iptablesInboundStaticRules is the list of iptables rules related to inbound traffic interception and redirection
var iptablesInboundStaticRules = []string{
	// Redirects inbound TCP traffic hitting the PROXY_IN_REDIRECT chain to Envoy's inbound listener port
	fmt.Sprintf("iptables -t nat -A PROXY_IN_REDIRECT -p tcp -j REDIRECT --to-port %d", constants.EnvoyInboundListenerPort),

	// For inbound traffic jump from PREROUTING chain to PROXY_INBOUND chain
	"iptables -t nat -A PREROUTING -p tcp -j PROXY_INBOUND",

	// Skip metrics query traffic being directed to Envoy's inbound prometheus listener port
	fmt.Sprintf("iptables -t nat -A PROXY_INBOUND -p tcp --dport %d -j RETURN", constants.EnvoyPrometheusInboundListenerPort),

	// Skip inbound health probes; These ports will be explicitly handled by listeners configured on the
	// Envoy proxy IF any health probes have been configured in the Pod Spec.
	// TODO(draychev): Do not add these if no health probes have been defined (https://github.com/openservicemesh/osm/issues/2243)
	fmt.Sprintf("iptables -t nat -A PROXY_INBOUND -p tcp --dport %d -j RETURN", livenessProbePort),
	fmt.Sprintf("iptables -t nat -A PROXY_INBOUND -p tcp --dport %d -j RETURN", readinessProbePort),
	fmt.Sprintf("iptables -t nat -A PROXY_INBOUND -p tcp --dport %d -j RETURN", startupProbePort),

	// Redirect remaining inbound traffic to Envoy
	"iptables -t nat -A PROXY_INBOUND -p tcp -j PROXY_IN_REDIRECT",
}

// generateIptablesCommands generates a list of iptables commands to set up sidecar interception and redirection
func generateIptablesCommands(outboundIPRangeExclusionList []string) []string {
	var cmd []string

	// 1. Create redirection chains
	cmd = append(cmd, iptablesRedirectionChains...)

	// 2. Create outbound rules
	cmd = append(cmd, iptablesOutboundStaticRules...)

	// 3. Create inbound rules
	cmd = append(cmd, iptablesInboundStaticRules...)

	// 4. Create dynamic outbound exclusion rules
	for _, cidr := range outboundIPRangeExclusionList {
		// *Note: it is important to use the insert option '-I' instead of the append option '-A' to ensure the exclusion
		// rules take precedence over the static redirection rules. Iptables rules are evaluated in order.
		rule := fmt.Sprintf("iptables -t nat -I PROXY_OUTPUT -d %s -j RETURN", cidr)
		cmd = append(cmd, rule)
	}

	return cmd
}
