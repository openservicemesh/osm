#!/bin/bash

set -aueo pipefail

REGISTRY=$(echo "$CTR_REGISTRY" | awk -F'.' '{print $1}')
REGISTRY_URL=$(echo "$CTR_REGISTRY" | awk -F'.' '{print $1 "." $2 "." $3}')

for ns in "$BOOKBUYER_NAMESPACE" "$BOOKSTORE_NAMESPACE" "$BOOKTHIEF_NAMESPACE"; do
    kubectl delete secrets "$CTR_REGISTRY_CREDS_NAME" -n "$ns" || true
    kubectl create secret docker-registry "$CTR_REGISTRY_CREDS_NAME" \
        -n "$ns" \
        --docker-server="$REGISTRY_URL" \
        --docker-username="$REGISTRY" \
        --docker-email="noone@example.com" \
        --docker-password="$DOCKER_PASS"
done
