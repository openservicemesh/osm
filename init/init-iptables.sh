#!/bin/bash

set -aueo pipefail

PROXY_ADMIN_PORT=${PROXY_ADMIN_PORT:-15000}
PROXY_STATS_PORT=${PROXY_STATS_PORT:-15010}
PROXY_PORT=${PROXY_PORT:-15001}
PROXY_INBOUND_PORT=${PROXY_INBOUND_PORT:-15003}
PROXY_UID=${PROXY_UID:-1337}
SSH_PORT=${SSH_PORT:-22}

# These ports are dedicated to Envoy listeners for the liveness, readiness, and startup probes
# TODO(draychev): Dynamically generate this bash file (https://github.com/openservicemesh/osm/issues/2243)
PROXY_LIVENESS_PROBE_PORT=${PROXY_LIVENESS_PROBE_PORT:-15901}
PROXY_READINESS_PROBE_PORT=${PROXY_READINESS_PROBE_PORT:-15902}
PROXY_STARTUP_PROBE_PORT=${PROXY_STARTUP_PROBE_PORT:-15903}

# Create a new chain for redirecting outbound traffic to PROXY_PORT
iptables -t nat -N PROXY_REDIRECT
iptables -t nat -A PROXY_REDIRECT -p tcp -j REDIRECT --to-port "${PROXY_PORT}"

# Traffic to the Proxy Admin port flows to the Proxy -- not redirected
iptables -t nat -A PROXY_REDIRECT -p tcp --dport "${PROXY_ADMIN_PORT}" -j ACCEPT


# Create a new chain for redirecting inbound traffic to PROXY_INBOUND_PORT
iptables -t nat -N PROXY_IN_REDIRECT
iptables -t nat -A PROXY_IN_REDIRECT -p tcp -j REDIRECT --to-port "${PROXY_INBOUND_PORT}"

# Create a new chain to redirect inbound traffic to Envoy
iptables -t nat -N PROXY_INBOUND
iptables -t nat -A PREROUTING -p tcp -j PROXY_INBOUND

# Skip inbound SSH redirection
iptables -t nat -A PROXY_INBOUND -p tcp --dport "${SSH_PORT}" -j RETURN

# Skip inbound stats query redirection
iptables -t nat -A PROXY_INBOUND -p tcp --dport "${PROXY_STATS_PORT}" -j RETURN

# Skip inbound health probes; These ports will be explicitly handled by listeners configured on the
# Envoy proxy IF any health probes have been configured in the Pod Spec.
# TODO(draychev): Do not add these if no health probes have been defined (https://github.com/openservicemesh/osm/issues/2243)
iptables -t nat -A PROXY_INBOUND -p tcp --dport "${PROXY_LIVENESS_PROBE_PORT}" -j RETURN
iptables -t nat -A PROXY_INBOUND -p tcp --dport "${PROXY_READINESS_PROBE_PORT}" -j RETURN
iptables -t nat -A PROXY_INBOUND -p tcp --dport "${PROXY_STARTUP_PROBE_PORT}" -j RETURN

# Redirect remaining inbound traffic to PROXY_INBOUND_PORT
iptables -t nat -A PROXY_INBOUND -p tcp -j PROXY_IN_REDIRECT

# Create a new chain to redirect outbound traffic to Envoy
iptables -t nat -N PROXY_OUTPUT

# For all TCP traffic, jump to PROXY_OUTPUT chain from OUTPUT chain
iptables -t nat -A OUTPUT -p tcp -j PROXY_OUTPUT

# TODO(shashank): Redirect app back calls to itself using PROXY_UID

# Don't redirect Envoy traffic back to itself for non-loopback traffic
iptables -t nat -A PROXY_OUTPUT -m owner --uid-owner "${PROXY_UID}" -j RETURN

# Skip localhost traffic
iptables -t nat -A PROXY_OUTPUT -d 127.0.0.1/32 -j RETURN

# Redirect remaining outbound traffic to Envoy
iptables -t nat -A PROXY_OUTPUT -j PROXY_REDIRECT
