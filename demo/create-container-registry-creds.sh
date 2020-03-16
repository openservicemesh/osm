#!/bin/bash

set -aueo pipefail

# shellcheck disable=SC1091
source .env

REGISTRY=$(echo "$CTR_REGISTRY" | awk -F'.' '{print $1}')
REGISTRY_URL=$(echo "$CTR_REGISTRY" | awk -F'.' '{print $1 "." $2 "." $3}')

echo "Creating container registry credentials ($CTR_REGISTRY_CREDS_NAME) for Kubernetes for the given Azure Container Registry ($REGISTRY_URL)"

DOCKER_PASSWORD=$(az acr credential show -n "$REGISTRY" --query "passwords[0].value" | tr -d '"')

for ns in "$BOOKBUYER_NAMESPACE" "$BOOKSTORE_NAMESPACE" "$BOOKTHIEF_NAMESPACE" "$K8S_NAMESPACE"; do
    kubectl delete secrets "$CTR_REGISTRY_CREDS_NAME" -n "$ns" || true
    kubectl create secret docker-registry "$CTR_REGISTRY_CREDS_NAME" \
        -n "$ns" \
        --docker-server="$REGISTRY_URL" \
        --docker-username="$REGISTRY" \
        --docker-email="noone@example.com" \
        --docker-password="$DOCKER_PASSWORD"
done
