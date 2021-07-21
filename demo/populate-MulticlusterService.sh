#!/bin/bash

set -auexo pipefail

# TODO: Parameterize this value: https://github.com/openservicemesh/osm/issues/3804
GATEWAY_PORT=14080

# Populate the IP of the Multicluster Gateway from the OTHER cluster
##########################
# Get IP addresses of OSM Multicluster Gateways
kubectl config use-context 'alpha'
ALPHA_OSM_GATEWAY_IP=$(kubectl get pods -n 'osm-system' --selector app=osm-multicluster-gateway -o json | jq -r '.items[].status.podIP')

kubectl config use-context 'beta'
BETA_OSM_GATEWAY_IP=$(kubectl get pods -n 'osm-system' --selector app=osm-multicluster-gateway -o json | jq -r '.items[].status.podIP')

##########################
# Populate Alpha w/ Beta's IP
kubectl config use-context 'alpha'
kubectl patch MulticlusterService \
        --namespace bookstore \
        bookstore \
        --type merge \
        --patch '{"spec":{"clusters":[{"name":"beta","address":"'"${BETA_OSM_GATEWAY_IP}"':'"${GATEWAY_PORT}"'"}]}}'

kubectl get MulticlusterService \
        --namespace bookstore bookstore \
        -o json | jq '.spec.clusters'


##########################
# Populate Beta w/ Alpha's IP
kubectl config use-context 'beta'
kubectl patch MulticlusterService \
        --namespace bookstore \
        bookstore \
        --type merge \
        --patch '{"spec":{"clusters":[{"name":"alpha","address":"'"${ALPHA_OSM_GATEWAY_IP}"':'"${GATEWAY_PORT}"'"}]}}'

kubectl get MulticlusterService \
        --namespace bookstore bookstore \
        -o json | jq '.spec.clusters'
