#!/bin/bash

set -auexo pipefail

PROXY_PORT=${PROXY_PORT:-15001}
PROXY_UID=${PROXY_UID:-1337}

iptables -t nat -N PROXY_REDIRECT

# Traffic to the Proxy Admin port flows to the Proxy -- not redirected!
iptables -t nat -A PROXY_REDIRECT -p tcp --dport 15000 -j ACCEPT
iptables -t nat -A PROXY_REDIRECT -p tcp -j REDIRECT --to-port "${PROXY_PORT}"
iptables -t nat -A PREROUTING -j PROXY_REDIRECT

iptables -t nat -N PROXY_OUTPUT
iptables -t nat -A OUTPUT -p tcp -j PROXY_OUTPUT
iptables -t nat -A PROXY_OUTPUT -m owner --uid-owner "${PROXY_UID}" -j RETURN
iptables -t nat -A PROXY_OUTPUT -p tcp --dport 8001 -j RETURN
iptables -t nat -A PROXY_OUTPUT -j PROXY_REDIRECT

exit 0
