#!/bin/bash

MULTICLUSTER_CONTEXTS="${MULTICLUSTER_CONTEXTS:-alpha beta}"


for CONTEXT in $MULTICLUSTER_CONTEXTS; do
    # shellcheck disable=SC2034
    #### BOOKSTORE_CLUSTER_ID="$CONTEXT" # this is used when deploying bookstore app
    unset BOOKSTORE_CLUSTER_ID

    kubectl config use-context "$CONTEXT"

    OSM_GATEWAY_IP=$(kubectl get pods -n 'osm-system' --selector app=osm-gateway -o json | jq -r '.items[].status.podIP')

    echo $CONTEXT $OSM_GATEWAY_IP

    kubectl patch MulticlusterService \
            --namespace bookstore \
            bookstore \
            --type merge \
            --patch '{"spec":{"clusters":[{"name":"'${CONTEXT}'","address":"'${OSM_GATEWAY_IP}':9876"}]}}'

    kubectl get MulticlusterService \
            --namespace bookstore bookstore \
            -o json | jq '.spec.clusters'
done
