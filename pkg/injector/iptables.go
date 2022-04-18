package injector

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/constants"
)

// iptablesInboundStaticRules is the list of iptables rules related to inbound traffic interception and redirection
var iptablesInboundStaticRules = []string{
	// Redirects inbound TCP traffic hitting the OSM_PROXY_IN_REDIRECT chain to Envoy's inbound listener port
	fmt.Sprintf("-A OSM_PROXY_IN_REDIRECT -p tcp -j REDIRECT --to-port %d", constants.EnvoyInboundListenerPort),

	// For inbound traffic jump from PREROUTING chain to OSM_PROXY_INBOUND chain
	"-A PREROUTING -p tcp -j OSM_PROXY_INBOUND",

	// Skip metrics query traffic being directed to Envoy's inbound prometheus listener port
	fmt.Sprintf("-A OSM_PROXY_INBOUND -p tcp --dport %d -j RETURN", constants.EnvoyPrometheusInboundListenerPort),

	// Skip inbound health probes; These ports will be explicitly handled by listeners configured on the
	// Envoy proxy IF any health probes have been configured in the Pod Spec.
	// TODO(draychev): Do not add these if no health probes have been defined (https://github.com/openservicemesh/osm/issues/2243)
	fmt.Sprintf("-A OSM_PROXY_INBOUND -p tcp --dport %d -j RETURN", livenessProbePort),
	fmt.Sprintf("-A OSM_PROXY_INBOUND -p tcp --dport %d -j RETURN", readinessProbePort),
	fmt.Sprintf("-A OSM_PROXY_INBOUND -p tcp --dport %d -j RETURN", startupProbePort),
	// Skip inbound health probes (originally TCPSocket health probes); requests handled by osm-healthcheck
	fmt.Sprintf("-A OSM_PROXY_INBOUND -p tcp --dport %d -j RETURN", healthcheckPort),

	// Redirect remaining inbound traffic to Envoy
	"-A OSM_PROXY_INBOUND -p tcp -j OSM_PROXY_IN_REDIRECT",
}

func genIPTablesOutboundStaticRules(cfg configurator.Configurator) []string {
	// iptablesOutboundStaticRules is the list of iptables rules related to outbound traffic interception and redirection
	iptablesOutboundStaticRules := []string{
		// Redirects outbound TCP traffic hitting OSM_PROXY_OUT_REDIRECT chain to Envoy's outbound listener port
		fmt.Sprintf("-A OSM_PROXY_OUT_REDIRECT -p tcp -j REDIRECT --to-port %d", constants.EnvoyOutboundListenerPort),

		// Traffic to the Proxy Admin port flows to the Proxy -- not redirected
		fmt.Sprintf("-A OSM_PROXY_OUT_REDIRECT -p tcp --dport %d -j ACCEPT", constants.EnvoyAdminPort),
	}

	iptablesOutboundStaticRules = append(iptablesOutboundStaticRules, []string{
		// For all other outbound TCP traffic jump from OUTPUT chain to OSM_PROXY_OUTBOUND chain
		"-A OUTPUT -p tcp -j OSM_PROXY_OUTBOUND",

		// Outbound traffic from Envoy to the local app over the loopback interface should jump to the inbound proxy redirect chain.
		// So when an app directs traffic to itself via the k8s service, traffic flows as follows:
		// app -> local envoy's outbound listener -> iptables -> local envoy's inbound listener -> app
		fmt.Sprintf("-A OSM_PROXY_OUTBOUND -o lo ! -d 127.0.0.1/32 -m owner --uid-owner %d -j OSM_PROXY_IN_REDIRECT", constants.EnvoyUID),

		// Outbound traffic from the app to itself over the loopback interface is not be redirected via the proxy.
		// E.g. when app sends traffic to itself via the pod IP.
		fmt.Sprintf("-A OSM_PROXY_OUTBOUND -o lo -m owner ! --uid-owner %d -j RETURN", constants.EnvoyUID),

		// Don't redirect Envoy traffic back to itself, return it to the next chain for processing
		fmt.Sprintf("-A OSM_PROXY_OUTBOUND -m owner --uid-owner %d -j RETURN", constants.EnvoyUID),

		// Skip localhost traffic, doesn't need to be routed via the proxy
		"-A OSM_PROXY_OUTBOUND -d 127.0.0.1/32 -j RETURN",
	}...)

	return iptablesOutboundStaticRules
}

// generateIptablesCommands generates a list of iptables commands to set up sidecar interception and redirection
func generateIptablesCommands(cfg configurator.Configurator, outboundIPRangeExclusionList []string, outboundIPRangeInclusionList []string, outboundPortExclusionList []int, inboundPortExclusionList []int) string {
	var rules strings.Builder

	fmt.Fprintln(&rules, `# OSM sidecar interception rules
*nat
:OSM_PROXY_INBOUND - [0:0]
:OSM_PROXY_IN_REDIRECT - [0:0]
:OSM_PROXY_OUTBOUND - [0:0]
:OSM_PROXY_OUT_REDIRECT - [0:0]`)
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
		rule := fmt.Sprintf("-I OSM_PROXY_INBOUND -p tcp --match multiport --dports %s -j RETURN", inboundPortsToExclude)
		cmds = append(cmds, rule)
	}

	iptablesOutboundStaticRules := genIPTablesOutboundStaticRules(cfg)

	// 3. Create outbound rules
	cmds = append(cmds, iptablesOutboundStaticRules...)

	//
	// Create outbound exclusion and inclusion rules.
	// *Note: exclusion rules must be applied before inclusions as order matters
	//

	// 4. Create dynamic outbound IP range exclusion rules
	for _, cidr := range outboundIPRangeExclusionList {
		// *Note: it is important to use the insert option '-I' instead of the append option '-A' to ensure the exclusion
		// rules take precedence over the static redirection rules. Iptables rules are evaluated in order.
		rule := fmt.Sprintf("-A OSM_PROXY_OUTBOUND -d %s -j RETURN", cidr)
		cmds = append(cmds, rule)
	}

	// 5. Create dynamic outbound ports exclusion rules
	if len(outboundPortExclusionList) > 0 {
		var portExclusionListStr []string
		for _, port := range outboundPortExclusionList {
			portExclusionListStr = append(portExclusionListStr, strconv.Itoa(port))
		}
		outboundPortsToExclude := strings.Join(portExclusionListStr, ",")
		rule := fmt.Sprintf("-A OSM_PROXY_OUTBOUND -p tcp --match multiport --dports %s -j RETURN", outboundPortsToExclude)
		cmds = append(cmds, rule)
	}

	// 6. Create dynamic outbound IP range inclusion rules
	if len(outboundIPRangeInclusionList) > 0 {
		// Redirect specified IP ranges to the proxy
		for _, cidr := range outboundIPRangeInclusionList {
			rule := fmt.Sprintf("-A OSM_PROXY_OUTBOUND -d %s -j OSM_PROXY_OUT_REDIRECT", cidr)
			cmds = append(cmds, rule)
		}
		// Remaining traffic not belonging to specified inclusion IP ranges are not redirected
		cmds = append(cmds, "-A OSM_PROXY_OUTBOUND -j RETURN")
	} else {
		// Redirect remaining outbound traffic to the proxy
		cmds = append(cmds, "-A OSM_PROXY_OUTBOUND -j OSM_PROXY_OUT_REDIRECT")
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
