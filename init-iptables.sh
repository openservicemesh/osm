#!/bin/bash

set -aueo pipefail

PROXY_ADMIN_PORT=${PROXY_ADMIN_PORT:-15000}
PROXY_STATS_PORT=${PROXY_STATS_PORT:-15010}
PROXY_PORT=${PROXY_PORT:-15001}
PROXY_INBOUND_PORT=${PROXY_INBOUND_PORT:-15003}
PROXY_UID=${PROXY_UID:-1337}
SSH_PORT=${SSH_PORT:-22}

# Create a new chain for redirecting outbound traffic to PROXY_PORT
iptables -t nat -N PROXY_REDIRECT

iptables -t nat -A PROXY_REDIRECT -p tcp --dport "2579" -j ACCEPT # kine
iptables -t nat -A PROXY_REDIRECT -p tcp --dport "5432" -j ACCEPT # postgres
iptables -t nat -A PROXY_REDIRECT -p tcp --dport "5556" -j ACCEPT # wsdex
iptables -t nat -A PROXY_REDIRECT -p tcp --dport "5557" -j ACCEPT # wsdex
iptables -t nat -A PROXY_REDIRECT -p tcp --dport "6000" -j ACCEPT # deviced
iptables -t nat -A PROXY_REDIRECT -p tcp --dport "6001" -j ACCEPT # deviced
iptables -t nat -A PROXY_REDIRECT -p tcp --dport "8081" -j ACCEPT # apiserver
iptables -t nat -A PROXY_REDIRECT -p tcp --dport "8100:8110" -j ACCEPT # proxyd
iptables -t nat -A PROXY_REDIRECT -p tcp --dport "8200" -j ACCEPT # valult
iptables -t nat -A PROXY_REDIRECT -p tcp --dport "8443" -j ACCEPT # apiserver
iptables -t nat -A PROXY_REDIRECT -p tcp --dport "9073" -j ACCEPT # identityd
iptables -t nat -A PROXY_REDIRECT -p tcp --dport "9085" -j ACCEPT # filed
iptables -t nat -A PROXY_REDIRECT -p tcp --dport "9097" -j ACCEPT # endpointd
iptables -t nat -A PROXY_REDIRECT -p tcp --dport "9122" -j ACCEPT # metrics
iptables -t nat -A PROXY_REDIRECT -p tcp --dport "10000" -j ACCEPT # radiusconfd
iptables -t nat -A PROXY_REDIRECT -p tcp --dport "32443" -j ACCEPT # sslport/apiserver


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

iptables -t nat -A PROXY_INBOUND -p tcp --dport "2579" -j RETURN  # kine
iptables -t nat -A PROXY_INBOUND -p tcp --dport "5432" -j RETURN  # postgres
iptables -t nat -A PROXY_INBOUND -p tcp --dport "5556" -j RETURN  # wsdex
iptables -t nat -A PROXY_INBOUND -p tcp --dport "5557" -j RETURN  # wsdex
iptables -t nat -A PROXY_INBOUND -p tcp --dport "6000" -j RETURN  # deviced
iptables -t nat -A PROXY_INBOUND -p tcp --dport "6001" -j RETURN  # deviced
iptables -t nat -A PROXY_INBOUND -p tcp --dport "8081" -j RETURN  # apiserver
iptables -t nat -A PROXY_INBOUND -p tcp --dport "8100:8110" -j RETURN  # proxyd
iptables -t nat -A PROXY_INBOUND -p tcp --dport "8200" -j RETURN  # valult
iptables -t nat -A PROXY_INBOUND -p tcp --dport "8443" -j RETURN  # apiserver
iptables -t nat -A PROXY_INBOUND -p tcp --dport "9073" -j RETURN  # identityd
iptables -t nat -A PROXY_INBOUND -p tcp --dport "9085" -j RETURN  # filed
iptables -t nat -A PROXY_INBOUND -p tcp --dport "9097" -j RETURN  # endpointd
iptables -t nat -A PROXY_INBOUND -p tcp --dport "9122" -j RETURN  # metricsd
iptables -t nat -A PROXY_INBOUND -p tcp --dport "10000" -j RETURN # radiusconfd
iptables -t nat -A PROXY_INBOUND -p tcp --dport "32443" -j RETURN # sslpoort/apiserver

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

#iptables -t nat -A PROXY_OUTPUT -p tcp --sport "9073" -j RETURN
#iptables -t nat -A PROXY_OUTPUT -p tcp --sport "8443" -j RETURN
#iptables -t nat -A PROXY_OUTPUT -p tcp --sport "8081" -j RETURN
#iptables -t nat -A PROXY_OUTPUT -p tcp --sport "2579" -j RETURN

# Redirect remaining outbound traffic to Envoy
iptables -t nat -A PROXY_OUTPUT -j PROXY_REDIRECT
