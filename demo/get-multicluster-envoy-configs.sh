#!/bin/bash

set -aueo pipefail

# This script automates the collection of Envoy config during Multicluster demo/development.
# The configs are collected from Envoys in the ALPHA kube context (cluster).

ALPHA_CLUSTER="${ALPHA_CLUSTER:-alpha}"
BETA_CLUSTER="${BETA_CLUSTER:-beta}"
MULTICLUSTER_CONTEXTS="${MULTICLUSTER_CONTEXTS:-$ALPHA_CLUSTER $BETA_CLUSTER}"

for CONTEXT in $MULTICLUSTER_CONTEXTS; do
    echo -e "\nSwitching context to ${CONTEXT}"
    kubectl config use-context "$CONTEXT" > /dev/null

    for NAMESPACE in bookbuyer bookstore; do
        for POD in $(kubectl get pod -n "${NAMESPACE}" --no-headers | awk '{print $1}'); do
            echo -e "Getting Envoy config from ${POD}"
            ./osm proxy get "config_dump?include_eds" "${POD}" --namespace="${NAMESPACE}" > "envoy_config_${CONTEXT}_${NAMESPACE}_${POD}.json"
        done
    done

    # Get the Envoy config of the OSM Multicluster Gateway pod
    NS='osm-system'
    PORT=15000
    TMP_PID_FILE=$(mktemp)
    OSM_GATEWAY_POD=$(kubectl get pod -n "${NS}" --selector app=osm-multicluster-gateway --no-headers | head -n1 | awk '{print $1}')
    echo -e "Getting Envoy config from Multicluster Gateway Pod: ${OSM_GATEWAY_POD}"

    ( kubectl port-forward -n "${NS}" "${OSM_GATEWAY_POD}" "${PORT}" > /dev/null & echo $! >&3 ) 3> "${TMP_PID_FILE}"
    sleep 1
    curl -s "http://localhost:${PORT}/config_dump?include_eds"  >  "envoy_config_${CONTEXT}_${NS}_${OSM_GATEWAY_POD}.json"
    kill -9 "$(<"${TMP_PID_FILE}")"
    rm -rf "${TMP_PID_FILE}"
done

echo -e "\nCollected Envoy config files:"
ls -lah envoy_config_*.json
