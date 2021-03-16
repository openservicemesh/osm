#!/bin/bash

set -x
set -aueo pipefail

PROXY_ADMIN_PORT=${PROXY_ADMIN_PORT:-15000}
PROXY_STATS_PORT=${PROXY_STATS_PORT:-15010}
PROXY_PORT=${PROXY_PORT:-15001}
PROXY_INBOUND_PORT=${PROXY_INBOUND_PORT:-15003}
PROXY_UID=${PROXY_UID:-1337}
SSH_PORT=${SSH_PORT:-22}

#clean old chain
iptables -t nat -F
iptables -t nat -X
#iptables -t nat -X PROXY_REDIRECT
#iptables -t nat -X PROXY_OUTPUT
#iptables -t nat -X PROXY_IN_REDIRECT
#iptables -t nat -X PROXY_INBOUND

# Create a new chain for redirecting outbound traffic to PROXY_PORT
iptables -t nat -N PROXY_REDIRECT

iptables -t nat -A PROXY_REDIRECT -p tcp --dport "22" -j ACCEPT # ssh port
iptables -t nat -A PROXY_REDIRECT -p tcp --dport "49" -j ACCEPT # tacacs
iptables -t nat -A PROXY_REDIRECT -p tcp --dport "69" -j ACCEPT # tftp
iptables -t nat -A PROXY_REDIRECT -p tcp --dport "88" -j ACCEPT # nacmode
iptables -t nat -A PROXY_REDIRECT -p tcp --dport "139" -j ACCEPT # samba
iptables -t nat -A PROXY_REDIRECT -p tcp --dport "389" -j ACCEPT # radius port
iptables -t nat -A PROXY_REDIRECT -p tcp --dport "443" -j ACCEPT # aruba
iptables -t nat -A PROXY_REDIRECT -p tcp --dport "445" -j ACCEPT # samba
iptables -t nat -A PROXY_REDIRECT -p tcp --dport "587" -j ACCEPT # email port
iptables -t nat -A PROXY_REDIRECT -p tcp --dport "636" -j ACCEPT # ldaps
iptables -t nat -A PROXY_REDIRECT -p tcp --dport "830" -j ACCEPT # netconf 
iptables -t nat -A PROXY_REDIRECT -p tcp --dport "2579" -j ACCEPT # kine
iptables -t nat -A PROXY_REDIRECT -p tcp --dport "2500" -j ACCEPT # osm-rest
iptables -t nat -A PROXY_REDIRECT -p tcp --dport "4343" -j ACCEPT # aruba
iptables -t nat -A PROXY_REDIRECT -p tcp --dport "5000" -j ACCEPT # devicedb
iptables -t nat -A PROXY_REDIRECT -p tcp --dport "5432" -j ACCEPT # postgres
iptables -t nat -A PROXY_REDIRECT -p tcp --dport "5556" -j ACCEPT # wsdex
iptables -t nat -A PROXY_REDIRECT -p tcp --dport "5557" -j ACCEPT # wsdex
iptables -t nat -A PROXY_REDIRECT -p tcp --dport "7201" -j ACCEPT # m3db/metricsd
iptables -t nat -A PROXY_REDIRECT -p tcp --dport "7203" -j ACCEPT # m3db/metricsd
iptables -t nat -A PROXY_REDIRECT -p tcp --dport "7301" -j ACCEPT # m3db/metricsd
iptables -t nat -A PROXY_REDIRECT -p tcp --dport "8005" -j ACCEPT # aws metricsd
iptables -t nat -A PROXY_REDIRECT -p tcp --dport "8080" -j ACCEPT # presto
iptables -t nat -A PROXY_REDIRECT -p tcp --dport "8081" -j ACCEPT # apiserver
iptables -t nat -A PROXY_REDIRECT -p tcp --dport "8100:8110" -j ACCEPT # proxyd
iptables -t nat -A PROXY_REDIRECT -p tcp --dport "8200" -j ACCEPT # valult
iptables -t nat -A PROXY_REDIRECT -p tcp --dport "8443" -j ACCEPT # apiserver
iptables -t nat -A PROXY_REDIRECT -p tcp --dport "9000:9004" -j ACCEPT # m3db/metricsd
iptables -t nat -A PROXY_REDIRECT -p tcp --dport "9053" -j ACCEPT # waves
iptables -t nat -A PROXY_REDIRECT -p tcp --dport "9063:9064" -j ACCEPT # alertdispatch
iptables -t nat -A PROXY_REDIRECT -p tcp --dport "9158" -j ACCEPT # alertruled
iptables -t nat -A PROXY_REDIRECT -p tcp --dport "9073" -j ACCEPT # identityd
iptables -t nat -A PROXY_REDIRECT -p tcp --dport "9083" -j ACCEPT # hive
iptables -t nat -A PROXY_REDIRECT -p tcp --dport "9085" -j ACCEPT # filed
iptables -t nat -A PROXY_REDIRECT -p tcp --dport "9092" -j ACCEPT # kafka
iptables -t nat -A PROXY_REDIRECT -p tcp --dport "9097" -j ACCEPT # endpointd
iptables -t nat -A PROXY_REDIRECT -p tcp --dport "9122" -j ACCEPT # metricsd
iptables -t nat -A PROXY_REDIRECT -p tcp --dport "9126" -j ACCEPT # nlp rest
iptables -t nat -A PROXY_REDIRECT -p tcp --dport "9128" -j ACCEPT # historyd
iptables -t nat -A PROXY_REDIRECT -p tcp --dport "9067" -j ACCEPT # rcad
iptables -t nat -A PROXY_REDIRECT -p tcp --dport "9200" -j ACCEPT # elastic
iptables -t nat -A PROXY_REDIRECT -p tcp --dport "9300" -j ACCEPT # elastic
iptables -t nat -A PROXY_REDIRECT -p tcp --dport "10000" -j ACCEPT # radiusconfd
iptables -t nat -A PROXY_REDIRECT -p tcp --dport "10080" -j ACCEPT # byod
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

iptables -t nat -A PROXY_INBOUND -p tcp --dport "49" -j RETURN  # tacacs
iptables -t nat -A PROXY_INBOUND -p tcp --dport "69" -j RETURN  # tftp
iptables -t nat -A PROXY_INBOUND -p tcp --dport "88" -j RETURN  # nacmode
iptables -t nat -A PROXY_INBOUND -p tcp --dport "139" -j RETURN  # samba
iptables -t nat -A PROXY_INBOUND -p tcp --dport "389" -j RETURN  # radius
iptables -t nat -A PROXY_INBOUND -p tcp --dport "443" -j RETURN  # aruba
iptables -t nat -A PROXY_INBOUND -p tcp --dport "445" -j RETURN  # samba
iptables -t nat -A PROXY_INBOUND -p tcp --dport "587" -j RETURN  # email
iptables -t nat -A PROXY_INBOUND -p tcp --dport "636" -j RETURN  # ldaps
iptables -t nat -A PROXY_INBOUND -p tcp --dport "830" -j RETURN  # netconf
iptables -t nat -A PROXY_INBOUND -p tcp --dport "2579" -j RETURN  # kine
iptables -t nat -A PROXY_INBOUND -p tcp --dport "2500" -j RETURN  # osm-rest
iptables -t nat -A PROXY_INBOUND -p tcp --dport "4343" -j RETURN  # aruba
iptables -t nat -A PROXY_INBOUND -p tcp --dport "5000" -j RETURN  # devicedb
iptables -t nat -A PROXY_INBOUND -p tcp --dport "5432" -j RETURN  # postgres
iptables -t nat -A PROXY_INBOUND -p tcp --dport "5556" -j RETURN  # wsdex
iptables -t nat -A PROXY_INBOUND -p tcp --dport "5557" -j RETURN  # wsdex
iptables -t nat -A PROXY_INBOUND -p tcp --dport "7201" -j RETURN  # m3db/metricsd
iptables -t nat -A PROXY_INBOUND -p tcp --dport "7203" -j RETURN  # m3db/metricsd
iptables -t nat -A PROXY_INBOUND -p tcp --dport "7301" -j RETURN  # m3db/metricsd
iptables -t nat -A PROXY_INBOUND -p tcp --dport "8005" -j RETURN  # aws metricsd
iptables -t nat -A PROXY_INBOUND -p tcp --dport "8080" -j RETURN  # presto
iptables -t nat -A PROXY_INBOUND -p tcp --dport "8081" -j RETURN  # apiserver
iptables -t nat -A PROXY_INBOUND -p tcp --dport "8100:8110" -j RETURN  # proxyd
iptables -t nat -A PROXY_INBOUND -p tcp --dport "8200" -j RETURN  # valult
iptables -t nat -A PROXY_INBOUND -p tcp --dport "8443" -j RETURN  # apiserver
iptables -t nat -A PROXY_INBOUND -p tcp --dport "9000:9004" -j RETURN  # m3db/metricsd
iptables -t nat -A PROXY_INBOUND -p tcp --dport "9053" -j RETURN  # waves
iptables -t nat -A PROXY_INBOUND -p tcp --dport "9063:9064" -j RETURN  # alertdispatch
iptables -t nat -A PROXY_INBOUND -p tcp --dport "9158" -j RETURN  # alertruled
iptables -t nat -A PROXY_INBOUND -p tcp --dport "9073" -j RETURN  # identityd
iptables -t nat -A PROXY_INBOUND -p tcp --dport "9083" -j RETURN  # hive
iptables -t nat -A PROXY_INBOUND -p tcp --dport "9085" -j RETURN  # filed
iptables -t nat -A PROXY_INBOUND -p tcp --dport "9092" -j RETURN  # kafka
iptables -t nat -A PROXY_INBOUND -p tcp --dport "9097" -j RETURN  # endpointd
iptables -t nat -A PROXY_INBOUND -p tcp --dport "9122" -j RETURN  # metricsd
iptables -t nat -A PROXY_INBOUND -p tcp --dport "9126" -j RETURN  # nlp rest
iptables -t nat -A PROXY_INBOUND -p tcp --dport "9128" -j RETURN  # historyd
iptables -t nat -A PROXY_INBOUND -p tcp --dport "9067" -j RETURN  # rcad
iptables -t nat -A PROXY_INBOUND -p tcp --dport "9200" -j RETURN  # elastic
iptables -t nat -A PROXY_INBOUND -p tcp --dport "9300" -j RETURN  # elastic
iptables -t nat -A PROXY_INBOUND -p tcp --dport "10000" -j RETURN # radiusconfd
iptables -t nat -A PROXY_INBOUND -p tcp --dport "10080" -j RETURN # byod
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

# Redirect pod and service-bound traffic to envoy
iptables -t nat -A PROXY_OUTPUT -d "${CIDR1}" -j PROXY_REDIRECT
iptables -t nat -A PROXY_OUTPUT -d "${CIDR2}" -j PROXY_REDIRECT

# skip remaining trafifc
iptables -t nat -A PROXY_OUTPUT -j RETURN # allow non-k8s traffic
