package injector

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/openservicemesh/osm/pkg/constants"
)

// iptablesOutboundStaticRules is the list of iptables rules related to outbound traffic interception and redirection
var iptablesOutboundStaticRules = []string{
	// Redirects outbound TCP traffic hitting PROXY_REDIRECT chain to Envoy's outbound listener port
	fmt.Sprintf("-A PROXY_REDIRECT -p tcp -j REDIRECT --to-port %d", constants.EnvoyOutboundListenerPort),

	// Traffic to the Proxy Admin port flows to the Proxy -- not redirected
	fmt.Sprintf("-A PROXY_REDIRECT -p tcp --dport %d -j ACCEPT", constants.EnvoyAdminPort),

	// For outbound TCP traffic jump from OUTPUT chain to PROXY_OUTPUT chain
	"-A OUTPUT -p tcp -j PROXY_OUTPUT",

	// Don't redirect Envoy traffic back to itself, return it to the next chain for processing
	fmt.Sprintf("-A PROXY_OUTPUT -m owner --uid-owner %d -j RETURN", constants.EnvoyUID),

	// Skip localhost traffic, doesn't need to be routed via the proxy
	"-A PROXY_OUTPUT -d 127.0.0.1/32 -j RETURN",

	// Redirect remaining outbound traffic to Envoy
	"-A PROXY_OUTPUT -j PROXY_REDIRECT",
}

// iptablesInboundStaticRules is the list of iptables rules related to inbound traffic interception and redirection
var iptablesInboundStaticRules = []string{
	// Redirects inbound TCP traffic hitting the PROXY_IN_REDIRECT chain to Envoy's inbound listener port
	fmt.Sprintf("-A PROXY_IN_REDIRECT -p tcp -j REDIRECT --to-port %d", constants.EnvoyInboundListenerPort),

	// For inbound traffic jump from PREROUTING chain to PROXY_INBOUND chain
	"-A PREROUTING -p tcp -j PROXY_INBOUND",

	// Skip metrics query traffic being directed to Envoy's inbound prometheus listener port
	fmt.Sprintf("-A PROXY_INBOUND -p tcp --dport %d -j RETURN", constants.EnvoyPrometheusInboundListenerPort),

	// Skip inbound health probes; These ports will be explicitly handled by listeners configured on the
	// Envoy proxy IF any health probes have been configured in the Pod Spec.
	// TODO(draychev): Do not add these if no health probes have been defined (https://github.com/openservicemesh/osm/issues/2243)
	fmt.Sprintf("-A PROXY_INBOUND -p tcp --dport %d -j RETURN", livenessProbePort),
	fmt.Sprintf("-A PROXY_INBOUND -p tcp --dport %d -j RETURN", readinessProbePort),
	fmt.Sprintf("-A PROXY_INBOUND -p tcp --dport %d -j RETURN", startupProbePort),

	// Redirect remaining inbound traffic to Envoy
	"-A PROXY_INBOUND -p tcp -j PROXY_IN_REDIRECT",
}

// generateIptablesCommands generates a list of iptables commands to set up sidecar interception and redirection
func generateIptablesCommands(outboundIPRangeExclusionList []string, outboundPortExclusionList []int, inboundPortExclusionList []int) string {
	var rules strings.Builder

	fmt.Fprintln(&rules, `# OSM sidecar interception rules
*nat
:PROXY_INBOUND - [0:0]
:PROXY_IN_REDIRECT - [0:0]
:PROXY_OUTPUT - [0:0]
:PROXY_REDIRECT - [0:0]`)
	var cmds []string

	// 1. Create inbound rules
	cmds = append(cmds, iptablesInboundStaticRules...)

	// 2. Create dynamic inbound ports exclusion rules
	if len(inboundPortExclusionList) > 0 {
		var portExclusionListStr []string
		for _, port := range inboundPortExclusionList {
			portExclusionListStr = append(portExclusionListStr, strconv.Itoa(port))
		}
		inboundPortsToExclude := strings.Join(portExclusionListStr, ",")
		rule := fmt.Sprintf("-I PROXY_INBOUND -p tcp --match multiport --dports %s -j RETURN", inboundPortsToExclude)
		cmds = append(cmds, rule)
	}

	// 3. Create outbound rules
	cmds = append(cmds, iptablesOutboundStaticRules...)

	// 4. Create dynamic outbound ip ranges exclusion rules
	for _, cidr := range outboundIPRangeExclusionList {
		// *Note: it is important to use the insert option '-I' instead of the append option '-A' to ensure the exclusion
		// rules take precedence over the static redirection rules. Iptables rules are evaluated in order.
		rule := fmt.Sprintf("-I PROXY_OUTPUT -d %s -j RETURN", cidr)
		cmds = append(cmds, rule)
	}

	// 5. Create dynamic outbound ports exclusion rules
	if len(outboundPortExclusionList) > 0 {
		var portExclusionListStr []string
		for _, port := range outboundPortExclusionList {
			portExclusionListStr = append(portExclusionListStr, strconv.Itoa(port))
		}
		outboundPortsToExclude := strings.Join(portExclusionListStr, ",")
		rule := fmt.Sprintf("-I PROXY_OUTPUT -p tcp --match multiport --dports %s -j RETURN", outboundPortsToExclude)
		cmds = append(cmds, rule)
	}

	for _, rule := range cmds {
		fmt.Fprintln(&rules, rule)
	}

	fmt.Fprint(&rules, "COMMIT")

	cmd := fmt.Sprintf(`iptables-restore --noflush <<EOF
%s
EOF
`, rules.String())

	return cmd
}
