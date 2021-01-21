package injector

import (
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"

	"github.com/openservicemesh/osm/pkg/constants"
)

// iptablesInitCommands is the list of iptables commands that need to be run in the given order
// to set up iptable redirection for inbound and outbound traffic via the sidecar proxy.
var iptablesInitCommands = []string{
	// Create a new chain for redirecting outbound traffic to PROXY_PORT
	"iptables -t nat -N PROXY_REDIRECT",
	fmt.Sprintf("iptables -t nat -A PROXY_REDIRECT -p tcp -j REDIRECT --to-port %d", constants.EnvoyOutboundListenerPort),

	// Traffic to the Proxy Admin port flows to the Proxy -- not redirected
	fmt.Sprintf("iptables -t nat -A PROXY_REDIRECT -p tcp --dport %d -j ACCEPT", constants.EnvoyAdminPort),

	// Create a new chain for redirecting inbound traffic to PROXY_INBOUND_PORT
	"iptables -t nat -N PROXY_IN_REDIRECT",
	fmt.Sprintf("iptables -t nat -A PROXY_IN_REDIRECT -p tcp -j REDIRECT --to-port %d", constants.EnvoyInboundListenerPort),

	// Create a new chain to redirect inbound traffic to Envoy
	"iptables -t nat -N PROXY_INBOUND",
	"iptables -t nat -A PREROUTING -p tcp -j PROXY_INBOUND",

	// Skip inbound stats query redirection
	fmt.Sprintf("iptables -t nat -A PROXY_INBOUND -p tcp --dport %d -j RETURN", constants.EnvoyPrometheusInboundListenerPort),

	// Skip inbound health probes; These ports will be explicitly handled by listeners configured on the
	// Envoy proxy IF any health probes have been configured in the Pod Spec.
	// TODO(draychev): Do not add these if no health probes have been defined (https://github.com/openservicemesh/osm/issues/2243)
	fmt.Sprintf("iptables -t nat -A PROXY_INBOUND -p tcp --dport %d -j RETURN", livenessProbePort),
	fmt.Sprintf("iptables -t nat -A PROXY_INBOUND -p tcp --dport %d -j RETURN", readinessProbePort),
	fmt.Sprintf("iptables -t nat -A PROXY_INBOUND -p tcp --dport %d -j RETURN", startupProbePort),

	// Redirect remaining inbound traffic to PROXY_INBOUND_PORT
	"iptables -t nat -A PROXY_INBOUND -p tcp -j PROXY_IN_REDIRECT",

	// Create a new chain to redirect outbound traffic to Envoy
	"iptables -t nat -N PROXY_OUTPUT",

	// For all TCP traffic, jump to PROXY_OUTPUT chain from OUTPUT chain
	"iptables -t nat -A OUTPUT -p tcp -j PROXY_OUTPUT",

	// TODO(shashank): Redirect app back calls to itself using PROXY_UID

	// Don't redirect Envoy traffic back to itself for non-loopback traffic
	fmt.Sprintf("iptables -t nat -A PROXY_OUTPUT -m owner --uid-owner %d -j RETURN", constants.EnvoyUID),

	// Skip localhost traffic
	"iptables -t nat -A PROXY_OUTPUT -d 127.0.0.1/32 -j RETURN",

	// Redirect remaining outbound traffic to Envoy
	"iptables -t nat -A PROXY_OUTPUT -j PROXY_REDIRECT",
}

func getInitContainerSpec(containerName, containerImage string) corev1.Container {
	iptablesInitCommands := strings.Join(iptablesInitCommands, " && ")

	return corev1.Container{
		Name:  containerName,
		Image: containerImage,
		SecurityContext: &corev1.SecurityContext{
			Capabilities: &corev1.Capabilities{
				Add: []corev1.Capability{
					"NET_ADMIN",
				},
			},
		},
		Command: []string{"/bin/sh"},
		Args: []string{
			"-c",
			iptablesInitCommands,
		},
	}
}
